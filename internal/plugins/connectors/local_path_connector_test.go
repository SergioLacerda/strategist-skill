package connectors_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSkillMD(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))
}

const validSkillMD = `---
name: sample-skill
description: A sample ORKA package for connector tests.
metadata:
  version: "1.0.0"
  author: test-author
---

# Sample Skill

Body content.
`

func TestResolveLocalPackageValidPackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, validSkillMD)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "references", "notes.md"), []byte("reference notes"), 0o644))

	pkg, err := connectors.ResolveLocalPackage(dir)
	require.NoError(t, err)
	assert.Equal(t, "sample-skill", pkg.ID)
	assert.Equal(t, "1.0.0", pkg.Version)
	assert.Equal(t, "test-author", pkg.Publisher)
	assert.Equal(t, "orka/v1", pkg.ManifestSchema)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, pkg.Digest)
	assert.Positive(t, pkg.ArtifactSize)
	assert.Equal(t, "file://"+dir, pkg.ArtifactURI)
}

func TestResolveLocalPackageDigestIsDeterministicAndOrderIndependent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, validSkillMD)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scripts", "run.sh"), []byte("#!/bin/sh\necho hi\n"), 0o644))

	first, err := connectors.ResolveLocalPackage(dir)
	require.NoError(t, err)
	second, err := connectors.ResolveLocalPackage(dir)
	require.NoError(t, err)
	assert.Equal(t, first.Digest, second.Digest)

	// A content change must change the digest.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scripts", "run.sh"), []byte("#!/bin/sh\necho changed\n"), 0o644))
	third, err := connectors.ResolveLocalPackage(dir)
	require.NoError(t, err)
	assert.NotEqual(t, first.Digest, third.Digest)
}

func TestResolveLocalPackageMissingSkillMDIsMalformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SKILL.md")
}

func TestResolveLocalPackageMissingFrontmatterIsMalformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, "# No frontmatter here\n")
	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frontmatter")
}

func TestResolveLocalPackageMissingNameIsMalformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, "---\ndescription: no name field\n---\nbody\n")
	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestResolveLocalPackageOversizedManifestIsRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	oversized := "---\nname: too-big\nmetadata:\n  version: \"1.0.0\"\n---\n" + strings.Repeat("x", domain.MaxPluginManifestBytes)
	writeSkillMD(t, dir, oversized)

	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestResolveLocalPackageFrontmatterMissingClosingDelimiterIsMalformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, "---\nname: sample\nno closing delimiter here\n")
	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing closing")
}

func TestResolveLocalPackageFrontmatterInvalidYAMLIsMalformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// A leading blank line before the opening delimiter is still accepted as a
	// valid frontmatter start, but the byte offset used to slice past the
	// delimiter no longer lines up with it, so the sliced frontmatter fails to
	// parse as YAML — exercising the "frontmatter: %w" wrapped-yaml-error path.
	writeSkillMD(t, dir, "\n---\nname: sample\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n")
	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frontmatter")
}

func TestResolveLocalPackageUnreadableSubdirectoryIsRejected(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}

	dir := t.TempDir()
	writeSkillMD(t, dir, validSkillMD)
	sub := filepath.Join(dir, "scripts")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "run.sh"), []byte("x"), 0o644))
	require.NoError(t, os.Chmod(sub, 0o000))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
}

func TestResolveLocalPackageOversizedPathIsRejected(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, validSkillMD)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	longName := strings.Repeat("a", domain.MaxPluginPathLength+10) + ".txt"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", longName), []byte("x"), 0o644))

	_, err := connectors.ResolveLocalPackage(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestLocalPathConnectorCapabilitiesNeverClaimInvoke(t *testing.T) {
	t.Parallel()

	connector := connectors.LocalPathConnector{ConnectorID: "local-path", ConnectorAPIVersion: "strategist-connector-api/1"}
	caps := connector.Capabilities(context.Background())
	assert.True(t, caps.CanResolve)
	assert.True(t, caps.CanProbe)
	assert.False(t, caps.CanInvoke)
	assert.False(t, caps.CanEnforcePermissions)
}

func TestLocalPathConnectorResolveReflectsPackageValidity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSkillMD(t, dir, validSkillMD)
	connector := connectors.LocalPathConnector{ConnectorID: "local-path"}

	ready := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: "sample-skill", Path: dir})
	assert.Equal(t, domain.ReadinessReady, ready.Status)

	blocked := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: "sample-skill", Path: t.TempDir()})
	assert.Equal(t, domain.ReadinessBlocked, blocked.Status)
	assert.Equal(t, "local_package_invalid", blocked.ReasonCode)
}

func TestLocalPathConnectorResolveBlocksIncompleteLocator(t *testing.T) {
	t.Parallel()

	connector := connectors.LocalPathConnector{ConnectorID: "local-path"}
	missingID := connector.Resolve(context.Background(), connectors.RuntimeLocator{Path: t.TempDir()})
	assert.Equal(t, domain.ReadinessBlocked, missingID.Status)
	assert.Equal(t, "locator_incomplete", missingID.ReasonCode)

	missingPath := connector.Resolve(context.Background(), connectors.RuntimeLocator{ID: "sample-skill"})
	assert.Equal(t, domain.ReadinessBlocked, missingPath.Status)
	assert.Equal(t, "locator_incomplete", missingPath.ReasonCode)
}

func TestLocalPathConnectorProbeValidatesStaticInputs(t *testing.T) {
	t.Parallel()

	connector := connectors.LocalPathConnector{ConnectorID: "local-path"}
	probe := connector.Probe(context.Background(), domain.InstalledInstance{ID: "sample-skill"}, "invoke")
	assert.Equal(t, domain.ReadinessUnknown, probe.Status)
	assert.Equal(t, "probe_not_verified", probe.ReasonCode)
	assert.Contains(t, probe.Detail, "no runtime invocation")

	missingInstance := connector.Probe(context.Background(), domain.InstalledInstance{}, "invoke")
	assert.Equal(t, domain.ReadinessBlocked, missingInstance.Status)
	assert.Equal(t, "probe_input_incomplete", missingInstance.ReasonCode)

	missingEntrypoint := connector.Probe(context.Background(), domain.InstalledInstance{ID: "sample-skill"}, "")
	assert.Equal(t, domain.ReadinessBlocked, missingEntrypoint.Status)
	assert.Equal(t, "probe_input_incomplete", missingEntrypoint.ReasonCode)
}

func TestLocalPathConnectorNeverClaimsInvokeOrRemove(t *testing.T) {
	t.Parallel()

	connector := connectors.LocalPathConnector{ConnectorID: "local-path"}
	invoked := connector.Invoke(context.Background(), connectors.InvocationEnvelope{Instance: domain.InstalledInstance{ID: "x"}, Entrypoint: "discover"})
	assert.Equal(t, domain.ReadinessUnsupported, invoked.Status)

	removed := connector.Remove(context.Background(), domain.InstalledInstance{ID: "x"})
	assert.Equal(t, domain.ReadinessUnsupported, removed.Status)

	observed := connector.Observe(context.Background(), domain.InstalledInstance{ID: "x"})
	assert.Equal(t, domain.ReadinessUnsupported, observed.Status)
	assert.Contains(t, observed.Enforcement.Limitations, "enforcement_not_supported")
}
