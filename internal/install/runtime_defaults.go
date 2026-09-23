package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/runtimefs"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"go.opentelemetry.io/otel/codes"
)

type runtimeDefaultPlan struct {
	embeddedHashes  map[string]string
	decisions       map[string]domain.RuntimeDefaultDecision
	levelingVersion int
	levelingDigest  string
}

func (s Service) planRuntimeDefaultUpgrade(ctx context.Context, strategistDir string, policy runtimeDefaultPolicy) (_ runtimeDefaultPlan, retErr error) {
	_, span := telemetry.Tracer().Start(ctx, "install.plan_runtime_defaults")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	embeddedHashes, err := s.embeddedNormativeHashes()
	if err != nil {
		return runtimeDefaultPlan{}, err
	}
	plan := runtimeDefaultPlan{
		embeddedHashes: embeddedHashes,
		decisions:      map[string]domain.RuntimeDefaultDecision{},
	}
	if err := s.populateLevelingAuthority(&plan); err != nil {
		return runtimeDefaultPlan{}, err
	}
	manifest, manifestLoaded, err := loadInstallManifest(strategistDir)
	if err != nil {
		return runtimeDefaultPlan{}, err
	}

	if err := populateRuntimeDefaultPlan(&plan, strategistDir, embeddedHashes, manifest, manifestLoaded, policy); err != nil {
		return runtimeDefaultPlan{}, err
	}

	return plan, nil
}

func (s Service) populateLevelingAuthority(plan *runtimeDefaultPlan) error {
	policy, err := s.embeddedLevelingPolicy()
	if err != nil {
		if skipOptionalLevelingError(s.Extractor, err) {
			return nil
		}
		return err
	}
	plan.levelingVersion = policy.Version
	plan.levelingDigest = policy.Digest()
	return nil
}

func (s Service) applyLevelingAuthority(manifest *domain.InstallManifest) error {
	policy, err := s.embeddedLevelingPolicy()
	if err != nil {
		if skipOptionalLevelingError(s.Extractor, err) {
			return nil
		}
		return err
	}
	manifest.LevelingPolicyVersion = policy.Version
	manifest.LevelingPolicyDigest = policy.Digest()
	return nil
}

func (s Service) embeddedLevelingPolicy() (leveling.Policy, error) {
	raw, err := s.Extractor.ReadFile("leveling.yaml")
	if err != nil {
		return leveling.Policy{}, fmt.Errorf("install: read embedded LEVELING policy: %w", err)
	}
	if len(raw) == 0 {
		return leveling.Policy{}, fmt.Errorf("install: embedded LEVELING policy is empty")
	}
	policy, err := leveling.Parse(raw)
	if err != nil {
		return leveling.Policy{}, fmt.Errorf("install: parse embedded LEVELING policy: %w", err)
	}
	return policy, nil
}

type levelingPolicyRequired interface {
	LevelingPolicyRequired() bool
}

func skipOptionalLevelingError(extractor domain.FileExtractor, err error) bool {
	if _, required := extractor.(levelingPolicyRequired); required {
		return false
	}
	return errors.Is(err, os.ErrNotExist)
}

func populateRuntimeDefaultPlan(plan *runtimeDefaultPlan, strategistDir string, embeddedHashes map[string]string, manifest domain.InstallManifest, manifestLoaded bool, policy runtimeDefaultPolicy) error {
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		decision, err := planRuntimeDefaultFile(strategistDir, file.Path, embeddedHashes[file.Path], manifest, manifestLoaded, policy)
		if err != nil {
			return err
		}
		if runtimeDefaultBlocksInstall(decision) {
			return fmt.Errorf("install: %s", domain.FormatRuntimeStaleDiagnostic(file.Path, decision))
		}
		plan.decisions[file.Path] = decision
	}
	return nil
}

func planRuntimeDefaultFile(
	strategistDir, relPath, embeddedHash string,
	manifest domain.InstallManifest,
	manifestLoaded bool,
	policy runtimeDefaultPolicy,
) (domain.RuntimeDefaultDecision, error) {
	runtimePath := filepath.Join(strategistDir, filepath.FromSlash(relPath))
	currentHash, exists, readErr := runtimefs.ReadSHA256(runtimePath)
	if readErr != nil {
		return "", fmt.Errorf("install: read normative runtime file %s: %w", relPath, readErr)
	}
	manifestFile, hasManifestEntry := manifest.FileByPath(relPath)
	return domain.DecideRuntimeDefaultUpdate(domain.RuntimeDefaultDecisionInput{
		Exists:          exists,
		CurrentHash:     currentHash,
		EmbeddedHash:    embeddedHash,
		ManifestHash:    manifestFile.SHA256,
		ManifestHistory: manifestFile.History,
		HasManifest:     manifestLoaded && hasManifestEntry,
		Force:           policy.Force,
		AllowDowngrade:  policy.AllowDowngrade,
	}), nil
}

func runtimeDefaultBlocksInstall(decision domain.RuntimeDefaultDecision) bool {
	return decision == domain.RuntimeDecisionConflict || decision == domain.RuntimeDecisionUnknownManifest ||
		decision == domain.RuntimeDecisionDowngrade
}

func (s Service) embeddedNormativeHashes() (map[string]string, error) {
	hashes := make(map[string]string, len(domain.NormativeRuntimeDefaultFiles()))
	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		data, err := s.Extractor.ReadFile(file.Path)
		if err != nil {
			return nil, fmt.Errorf("install: read embedded normative default %s: %w", file.Path, err)
		}
		hashes[file.Path] = domain.SHA256Hex(data)
	}
	return hashes, nil
}

func (s Service) applyRuntimeDefaultPlan(ctx context.Context, strategistDir string, plan runtimeDefaultPlan) (retErr error) {
	_, span := telemetry.Tracer().Start(ctx, "install.apply_runtime_defaults")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	for _, file := range domain.NormativeRuntimeDefaultFiles() {
		if err := s.applyRuntimeDefaultFile(strategistDir, file.Path, plan.decisions[file.Path]); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) applyRuntimeDefaultFile(strategistDir, relPath string, decision domain.RuntimeDefaultDecision) error {
	if decision == domain.RuntimeDecisionKeepCurrent {
		return nil
	}
	data, err := s.Extractor.ReadFile(relPath)
	if err != nil {
		return fmt.Errorf("install: read embedded normative default %s: %w", relPath, err)
	}
	targetPath := filepath.Join(strategistDir, filepath.FromSlash(relPath))
	if err := runtimefs.WriteFile(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("install: write normative default %s: %w", relPath, err)
	}
	return nil
}
