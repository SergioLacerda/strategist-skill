package install

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

// sidecarProvenance is the only source of upstream_* and license values, and
// of the openspec_root runtime block. Unknown keys are rejected.
type sidecarProvenance struct {
	UpstreamRepo          string                `yaml:"upstream_repo,omitempty"`
	UpstreamSkillPath     string                `yaml:"upstream_skill_path,omitempty"`
	UpstreamVersion       string                `yaml:"upstream_version,omitempty"`
	UpstreamCommit        string                `yaml:"upstream_commit,omitempty"`
	UpstreamContentDigest string                `yaml:"upstream_content_digest,omitempty"`
	License               string                `yaml:"license,omitempty"`
	Runtime               *domain.WeaponRuntime `yaml:"runtime,omitempty"`
}

// scaffoldInputs is the validated declaration plus what was read from the package.
type scaffoldInputs struct {
	version       string
	skillMD       []byte
	roles         []string
	slots         []string
	riskScore     string
	runtime       domain.WeaponRuntime
	scratchRoot   string
	provenance    sidecarProvenance
	provenanceRaw []byte
}

func loadScaffoldInputs(opts SidecarScaffoldOptions) (scaffoldInputs, error) {
	skillMD, version, err := readScaffoldPackage(opts.PackageDir, opts.Version)
	if err != nil {
		return scaffoldInputs{}, err
	}
	riskScore, err := validateScaffoldDeclaration(opts.Roles, opts.Slots)
	if err != nil {
		return scaffoldInputs{}, err
	}
	provenance, provenanceRaw, err := readScaffoldProvenance(opts.ProvenancePath)
	if err != nil {
		return scaffoldInputs{}, err
	}
	runtime, scratchRoot, err := resolveScaffoldRuntime(opts, provenance)
	if err != nil {
		return scaffoldInputs{}, err
	}
	return scaffoldInputs{
		version: version, skillMD: skillMD, roles: opts.Roles, slots: opts.Slots, riskScore: riskScore,
		runtime: runtime, scratchRoot: scratchRoot, provenance: provenance, provenanceRaw: provenanceRaw,
	}, nil
}

// readScaffoldPackage reads SKILL.md and resolves the version: the package's own
// metadata.version wins; declared is the operator's explicit fallback.
func readScaffoldPackage(dir, declared string) (skillMD []byte, version string, err error) {
	skillMD, err = os.ReadFile(filepath.Join(dir, "SKILL.md")) //nolint:gosec // G304: operator-declared package directory
	if err != nil {
		return nil, "", scaffoldError("skill_md_missing", "%s: %v", filepath.Join(dir, "SKILL.md"), err)
	}
	pkg, err := connectors.ResolveEmbeddedPackage(dir)
	if err != nil {
		return nil, "", scaffoldError("skill_md_invalid", "%v", err)
	}
	version = pkg.Version
	if version == "" {
		version = declared
	}
	if version == "" {
		return nil, "", scaffoldError("version_missing", "SKILL.md declares no metadata.version; pass --version")
	}
	return skillMD, version, nil
}

func readScaffoldProvenance(path string) (sidecarProvenance, []byte, error) {
	if path == "" {
		return sidecarProvenance{}, nil, nil
	}
	raw, err := os.ReadFile(path) //nolint:gosec // G304: operator-declared provenance file
	if err != nil {
		return sidecarProvenance{}, nil, scaffoldError("provenance_invalid", "read %s: %v", path, err)
	}
	var provenance sidecarProvenance
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&provenance); err != nil {
		return sidecarProvenance{}, nil, scaffoldError("provenance_invalid", "%s: %v", path, err)
	}
	return provenance, raw, nil
}

func resolveScaffoldRuntime(opts SidecarScaffoldOptions, provenance sidecarProvenance) (domain.WeaponRuntime, string, error) {
	switch opts.Runtime {
	case "", domain.RankedRuntimeEmbedded:
		return domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}, "", nil
	case domain.RankedRuntimeOpenSpecRoot:
		return resolveOpenSpecScaffoldRuntime(opts.PackageDir, provenance)
	default:
		return domain.WeaponRuntime{}, "", scaffoldError("runtime_unsupported", "runtime %q cannot be generated; use embedded or openspec_root, or author the sidecar by hand", opts.Runtime)
	}
}

func resolveOpenSpecScaffoldRuntime(dir string, provenance sidecarProvenance) (domain.WeaponRuntime, string, error) {
	if !isDir(filepath.Join(dir, "runtime")) || !isFile(filepath.Join(dir, "runtime.lock.yaml")) {
		return domain.WeaponRuntime{}, "", scaffoldError("runtime_bundle_missing", "openspec_root needs runtime/ and runtime.lock.yaml in %s", dir)
	}
	if provenance.Runtime == nil {
		return domain.WeaponRuntime{}, "", scaffoldError("runtime_declaration_missing", "the --provenance file must declare the runtime: block for openspec_root")
	}
	runtime := *provenance.Runtime
	if runtime.Kind != domain.RankedRuntimeOpenSpecRoot {
		return domain.WeaponRuntime{}, "", scaffoldError("provenance_invalid", "runtime.kind must be %q, got %q", domain.RankedRuntimeOpenSpecRoot, runtime.Kind)
	}
	if err := runtime.Validate(); err != nil {
		return domain.WeaponRuntime{}, "", scaffoldError("provenance_invalid", "runtime: %v", err)
	}
	return runtime, "runtime", nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
