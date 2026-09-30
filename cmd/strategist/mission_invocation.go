package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"gopkg.in/yaml.v3"
)

const invocationLifetime = 5 * time.Minute

func missionInvocationDependencies() missionadapter.InvocationDependencies {
	return missionadapter.InvocationDependencies{
		RootFlag: cliutilFlagRoot(), RequireMissionID: requireMissionID,
		ResolveBasePath: resolveActiveBasePathForMission,
		LoadMission: func(root, missionID string) (domain.MissionEngineStatus, error) {
			_, status, err := loadMission(root, missionID)
			return status, err
		},
		Build: buildMissionInvocation, Complete: completeMissionInvocation, ExecuteHost: executeMissionHost,
		WriteResult: writeMissionResult, ReadCompletion: readMissionCompletion,
	}
}

func cliutilFlagRoot() string { return cliutil.FlagRoot }

func resolveActiveBasePathForMission(root string) (string, string, error) {
	strategistRoot, basePath, err := cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve active base path: %w", err)
	}
	return strategistRoot, basePath, nil
}

func buildMissionInvocation(ctx context.Context, input missionadapter.InvocationBuildInput) (domain.MissionInvocationRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.MissionInvocationRequest{}, fmt.Errorf("build mission invocation: context: %w", err)
	}
	binding, weapon, payload, sourceDigest, err := resolveMissionInvocationWeapon(input)
	if err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	request, now, err := newMissionInvocationRequest(input, binding, weapon, payload, sourceDigest)
	if err != nil {
		return domain.MissionInvocationRequest{}, err
	}
	store := missionruntime.NewInvocationStore(input.Root)
	if err := store.Put(domain.MissionInvocationRecord{Request: request, CreatedAt: now, ExpiresAt: now.Add(invocationLifetime)}); err != nil {
		return domain.MissionInvocationRequest{}, fmt.Errorf("persist mission invocation: %w", err)
	}
	return request, nil
}

func resolveMissionInvocationWeapon(input missionadapter.InvocationBuildInput) (domain.RoleWeaponBinding, domain.CompiledWeapon, []byte, string, error) {
	active, lock, registry, err := loadMissionInvocationState(input.Root)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", err
	}
	binding, weapon, err := resolveEmbeddedInvocationBinding(active, lock, registry, input)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", err
	}
	payload, sourceDigest, err := (strategistembed.Extractor{}).ReadEmbeddedWeaponPayload(binding.WeaponID, binding.WeaponVersion)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", fmt.Errorf("embedded_payload_mismatch: %w", err)
	}
	if sourceDigest != binding.SourceDigest || sourceDigest != weapon.SourceDigest {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", fmt.Errorf("embedded_payload_mismatch: payload digest does not match compiled binding")
	}
	return binding, weapon, payload, sourceDigest, nil
}

func resolveEmbeddedInvocationBinding(active domain.ActiveConfig, lock domain.PluginLockFile, registry domain.CompiledRegistry, input missionadapter.InvocationBuildInput) (domain.RoleWeaponBinding, domain.CompiledWeapon, error) {
	binding, err := domain.ResolveRoleWeaponBinding(active, lock, registry, input.Role, input.Slot)
	if err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, fmt.Errorf("role_invocation_failed: %w", err)
	}
	if err := validateEmbeddedInvocationBinding(binding); err != nil {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, err
	}
	weapon, ok := registry.Weapon(binding.WeaponID, binding.WeaponVersion)
	if !ok {
		return domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, fmt.Errorf("compiled_binding_missing: Weapon %q is not in the compiled registry", domain.WeaponIdentity(binding.WeaponID, binding.WeaponVersion))
	}
	return binding, weapon, nil
}

func newMissionInvocationRequest(input missionadapter.InvocationBuildInput, binding domain.RoleWeaponBinding, weapon domain.CompiledWeapon, payload []byte, sourceDigest string) (domain.MissionInvocationRequest, time.Time, error) {
	requestID, err := newInvocationID()
	if err != nil {
		return domain.MissionInvocationRequest{}, time.Time{}, err
	}
	now := time.Now().UTC()
	requestInput := map[string]any{}
	if strings.TrimSpace(input.RequestContext) != "" {
		requestInput["request_context"] = input.RequestContext
	}
	return domain.MissionInvocationRequest{
		Protocol: domain.MissionInvocationProtocolVersion, RequestID: requestID,
		MissionID: input.MissionID, Role: input.Role, Slot: input.Slot,
		Weapon:        domain.MissionWeaponIdentity{ID: weapon.ID, Version: weapon.Version, Digest: weapon.Digest},
		BindingDigest: binding.BindingDigest, SourceDigest: sourceDigest,
		ExecutionMode: binding.ExecutionMode, Entrypoint: binding.Entrypoint,
		Payload: string(payload), Input: requestInput,
	}, now, nil
}

func validateEmbeddedInvocationBinding(binding domain.RoleWeaponBinding) error {
	if binding.Mode != domain.SlotBindingModeRanked {
		return fmt.Errorf("role_invocation_failed: mission invoke only supports Ranked Embedded prompt-bridge bindings")
	}
	if binding.RuntimeKind == domain.RankedRuntimeOpenSpecRoot {
		return fmt.Errorf("role_invocation_failed: Ranked openspec_root bindings execute through their declared private runtime and mission normalize-openspec, not mission invoke")
	}
	if binding.RuntimeKind != domain.RankedRuntimeEmbedded {
		return fmt.Errorf("role_invocation_failed: mission invoke only supports runtime kind %q, got %q", domain.RankedRuntimeEmbedded, binding.RuntimeKind)
	}
	return nil
}

func loadMissionInvocationState(root string) (domain.ActiveConfig, domain.PluginLockFile, domain.CompiledRegistry, error) {
	activeRaw, err := os.ReadFile(filepath.Join(root, "active.yaml")) //nolint:gosec // G304: root is resolved by the CLI runtime boundary.
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("read active.yaml: %w", err)
	}
	var active domain.ActiveConfig
	if err := yaml.Unmarshal(activeRaw, &active); err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("parse active.yaml: %w", err)
	}
	lockRaw, err := os.ReadFile(filepath.Join(root, "plugins.lock")) //nolint:gosec // G304: root is resolved by the CLI runtime boundary.
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(lockRaw, &lock); err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("parse plugins.lock: %w", err)
	}
	catalogRaw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: root is resolved by the CLI runtime boundary.
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("read compiled catalog: %w", err)
	}
	registry, err := domain.ParseCompiledRegistryCatalog(catalogRaw)
	if err != nil {
		return domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, fmt.Errorf("parse compiled catalog: %w", err)
	}
	return active, lock, registry, nil
}

func newInvocationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("mission invocation: generate request id: %w", err)
	}
	return "inv_" + hex.EncodeToString(raw[:]), nil
}
