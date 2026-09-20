package runtimepayload

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	skillDir         = "skills/openspec-propose"
	openSpecTreeDir  = skillDir + "/runtime"
	runtimeLockFile  = skillDir + "/runtime.lock.yaml"
	nodePayloadDir   = "nodepayload"
	embeddedProvider = "openspec-propose"
)

type runtimeLock struct {
	Node struct {
		Version string `yaml:"version"`
	} `yaml:"node"`
}

type openSpecBuildInfo struct {
	Version    string `yaml:"version"`
	Bundle     string `yaml:"bundle"`
	TreeSHA256 string `yaml:"tree_sha256"`
	TreeBytes  int64  `yaml:"tree_bytes"`
}

type nodeInfo struct {
	Version string `yaml:"version"`
	Target  string `yaml:"target"`
	File    string `yaml:"file"`
	Format  string `yaml:"format"`
	SHA256  string `yaml:"sha256"`
	Size    int64  `yaml:"size"`
}

// BuildEmbedded composes the runtime payload of one build target: the prebuilt
// OpenSpec tree that ships in the skill package (defaults) and the target's
// Node archive (nodeFS, laid out as nodepayload/<target>/...). Every pin comes
// from committed files or from files the build generated and digest-checked,
// and any skew between them is an error: a broken payload must never register.
func BuildEmbedded(defaults, nodeFS fs.FS, target string) (Manifest, fs.FS, error) {
	goos, goarch, ok := strings.Cut(target, "-")
	if !ok {
		return Manifest{}, nil, fmt.Errorf("runtime payload target %q must be <os>-<arch>", target)
	}
	tree, node, err := readBuildInputs(defaults, nodeFS, target)
	if err != nil {
		return Manifest{}, nil, err
	}
	m := embeddedManifest(goos, goarch, target, tree, node)
	if err := m.Validate(); err != nil {
		return Manifest{}, nil, err
	}
	return m, layeredFS{defaults: defaults, node: nodeFS}, nil
}

// readBuildInputs loads the pinned inputs and rejects any skew between them.
func readBuildInputs(defaults, nodeFS fs.FS, target string) (openSpecBuildInfo, nodeInfo, error) {
	var lock runtimeLock
	var tree openSpecBuildInfo
	var node nodeInfo
	if err := errors.Join(
		readYAML(defaults, runtimeLockFile, &lock),
		readYAML(defaults, openSpecTreeDir+"/"+BuildInfoFile, &tree),
		readYAML(nodeFS, path.Join(nodePayloadDir, target, "node.info.yaml"), &node),
	); err != nil {
		return tree, node, err
	}
	return tree, node, checkNodeSkew(lock, node, target)
}

func checkNodeSkew(lock runtimeLock, node nodeInfo, target string) error {
	if node.Version != lock.Node.Version {
		return fmt.Errorf("runtime payload node version %s does not match runtime.lock.yaml %s", node.Version, lock.Node.Version)
	}
	if node.Target != target {
		return fmt.Errorf("runtime payload was built for %s, not %s", node.Target, target)
	}
	return nil
}

func embeddedManifest(goos, goarch, target string, tree openSpecBuildInfo, node nodeInfo) Manifest {
	return Manifest{
		SchemaVersion: SchemaVersion,
		Provider:      embeddedProvider,
		Launcher: Launcher{
			Node:   map[string]string{"default": "node/bin/node", "windows": "node/node.exe"},
			Script: "openspec/" + tree.Bundle,
		},
		Components: []Component{
			{Name: "node", Version: node.Version, OS: goos, Arch: goarch, File: path.Join(nodePayloadDir, target, node.File), Format: node.Format, SHA256: node.SHA256, Size: node.Size, Dest: "node"},
			{Name: "openspec", Version: tree.Version, OS: AnyTarget, Arch: AnyTarget, File: openSpecTreeDir, Format: FormatDir, SHA256: tree.TreeSHA256, Size: tree.TreeBytes, Dest: "openspec"},
		},
	}
}

func readYAML(src fs.FS, name string, into any) error {
	raw, err := fs.ReadFile(src, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrPayloadMissing, name)
		}
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := yaml.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}

// layeredFS serves nodepayload/... from the per-target Node payload and every
// other path from the embedded skill defaults.
type layeredFS struct{ defaults, node fs.FS }

func (l layeredFS) Open(name string) (fs.File, error) {
	if name == nodePayloadDir || strings.HasPrefix(name, nodePayloadDir+"/") {
		return l.node.Open(name) //nolint:wrapcheck // fs.FS contract: callers expect the underlying *fs.PathError
	}
	return l.defaults.Open(name) //nolint:wrapcheck // fs.FS contract: callers expect the underlying *fs.PathError
}
