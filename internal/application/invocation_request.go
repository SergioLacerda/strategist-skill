package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// InvocationRequestInput contains the already-resolved immutable runtime
// facts needed to issue a host request. Registry and payload resolution stay
// in the composition root; request construction belongs to the application
// boundary and does not depend on Cobra or package main.
type InvocationRequestInput struct {
	Root                      string
	BasePath                  string
	MissionID                 string
	Role                      string
	Slot                      string
	RequestContext            string
	Binding                   domain.RoleWeaponBinding
	Weapon                    domain.CompiledWeapon
	Payload                   []byte
	SourceDigest              string
	ApprovalGatePackageDigest string
	ExecutionContract         string
	OutputContract            string
}

// NewInvocationRequest creates the immutable request envelope and its issue
// time. It is deliberately independent of host process execution.
func NewInvocationRequest(ctx context.Context, input InvocationRequestInput) (domain.MissionInvocationRequest, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return domain.MissionInvocationRequest{}, time.Time{}, fmt.Errorf("build mission invocation: context: %w", err)
	}
	requestID, err := NewInvocationID()
	if err != nil {
		return domain.MissionInvocationRequest{}, time.Time{}, err
	}
	now := time.Now().UTC()
	requestInput := map[string]any{}
	if input.ExecutionContract != "" {
		requestInput["execution_contract"] = input.ExecutionContract
	}
	if input.OutputContract != "" {
		requestInput["output_contract"] = input.OutputContract
	}
	if input.Role == "sniper" && input.Slot == string(domain.SlotExecution) {
		requestInput["refined_package"] = filepath.ToSlash(filepath.Join(input.BasePath, "refined", input.MissionID))
		requestInput["report_path"] = filepath.ToSlash(filepath.Join(input.BasePath, "archived", input.MissionID+"-report.md"))
		requestInput["approval_gate_package_digest"] = input.ApprovalGatePackageDigest
	}
	if strings.TrimSpace(input.RequestContext) != "" {
		requestInput["request_context"] = input.RequestContext
	}
	return domain.MissionInvocationRequest{
		Protocol: domain.MissionInvocationProtocolVersion, RequestID: requestID,
		MissionID: input.MissionID, Role: input.Role, Slot: input.Slot,
		Weapon:        domain.MissionWeaponIdentity{ID: input.Weapon.ID, Version: input.Weapon.Version, Digest: input.Weapon.Digest},
		BindingDigest: input.Binding.BindingDigest, SourceDigest: input.SourceDigest,
		ExecutionMode: input.Binding.ExecutionMode, Entrypoint: input.Binding.Entrypoint,
		Payload: string(input.Payload), Input: requestInput, Nonce: NewPromptNonce(),
	}, now, nil
}

// NewInvocationID returns an opaque, collision-resistant request identifier.
func NewInvocationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("mission invocation: generate request id: %w", err)
	}
	return "inv_" + hex.EncodeToString(raw[:]), nil
}

// NewPromptNonce returns the non-authenticating delimiter used by the host
// bridge to correlate a request and its echoed receipt.
func NewPromptNonce() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Errorf("read random prompt nonce: %w", err))
	}
	return hex.EncodeToString(raw)
}
