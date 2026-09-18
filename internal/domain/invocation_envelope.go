package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// InvocationEnvelopeSchemaVersion identifies the stable envelope schema.
const InvocationEnvelopeSchemaVersion = "strategist-invocation-envelope/v1"

// InvocationComponent is a runtime-readable reference selected for one phase.
// Content is intentionally referenced by digest instead of copied into the
// envelope; the agent/provider remains responsible for reading it.
type InvocationComponent struct {
	Ref      string        `json:"ref"`
	Kind     string        `json:"kind"`
	Phase    PipelinePhase `json:"phase,omitempty"`
	Digest   string        `json:"digest"`
	Selected bool          `json:"selected"`
}

// InvocationEnvelope is the deterministic structural context passed to a
// phase invocation. Components from future phases are never included.
type InvocationEnvelope struct {
	SchemaVersion       string                `json:"schema_version"`
	MissionID           string                `json:"mission_id"`
	Phase               PipelinePhase         `json:"phase"`
	Role                string                `json:"role"`
	Slot                string                `json:"slot"`
	Provider            string                `json:"provider"`
	RequiredContextRefs []string              `json:"required_context_refs"`
	OutputSchemaRef     string                `json:"output_schema_ref"`
	Components          []InvocationComponent `json:"components"`
	Fingerprint         string                `json:"fingerprint"`
	Plan                RoleInvocationPlan    `json:"plan"`
}

// ComposeInvocationRequest supplies the structural inputs to the composer.
type ComposeInvocationRequest struct {
	MissionID       string
	Phase           PipelinePhase
	Plan            RoleInvocationPlan
	OutputSchemaRef string
	Components      []InvocationComponent
}

// ComposeInvocationEnvelope selects and orders current-phase components and
// produces a content-addressed fingerprint. Empty component phases are global
// references and are included for every phase.
func ComposeInvocationEnvelope(request ComposeInvocationRequest) (InvocationEnvelope, error) {
	outputSchema, err := validateInvocationRequest(request)
	if err != nil {
		return InvocationEnvelope{}, err
	}
	components := selectedInvocationComponents(request.Phase, request.Components)

	refs := make([]string, 0, len(components))
	for _, component := range components {
		refs = append(refs, component.Ref)
	}
	plan := request.Plan
	plan.RequiredContextRefs = refs
	plan.OutputSchemaRef = outputSchema
	envelope := InvocationEnvelope{
		SchemaVersion:       InvocationEnvelopeSchemaVersion,
		MissionID:           request.MissionID,
		Phase:               request.Phase,
		Role:                request.Plan.Role,
		Slot:                request.Plan.Slot,
		Provider:            request.Plan.WeaponID,
		RequiredContextRefs: refs,
		OutputSchemaRef:     outputSchema,
		Components:          components,
		Plan:                plan,
	}
	fingerprint, err := invocationEnvelopeFingerprint(envelope)
	if err != nil {
		return InvocationEnvelope{}, err
	}
	envelope.Fingerprint = fingerprint
	return envelope, nil
}

func validateInvocationRequest(request ComposeInvocationRequest) (string, error) {
	if request.MissionID == "" {
		return "", fmt.Errorf("invocation envelope: mission id is required")
	}
	if request.Phase == "" {
		return "", fmt.Errorf("invocation envelope: phase is required")
	}
	if request.Plan.Role == "" || request.Plan.Slot == "" || request.Plan.WeaponID == "" {
		return "", fmt.Errorf("invocation envelope: role, slot, and provider are required")
	}
	outputSchema := request.OutputSchemaRef
	if outputSchema == "" {
		outputSchema = request.Plan.OutputSchemaRef
	}
	if outputSchema == "" {
		return "", fmt.Errorf("invocation envelope: output schema is required")
	}
	return outputSchema, nil
}

func selectedInvocationComponents(phase PipelinePhase, input []InvocationComponent) []InvocationComponent {
	components := make([]InvocationComponent, 0, len(input))
	for _, component := range input {
		if selectableInvocationComponent(phase, component) {
			components = append(components, component)
		}
	}
	sort.Slice(components, func(i, j int) bool {
		return invocationComponentLess(components[i], components[j])
	})
	return components
}

func selectableInvocationComponent(phase PipelinePhase, component InvocationComponent) bool {
	if !component.Selected || component.Ref == "" || component.Kind == "" || component.Digest == "" {
		return false
	}
	return component.Phase == "" || component.Phase == phase
}

func invocationComponentLess(left, right InvocationComponent) bool {
	if left.Ref != right.Ref {
		return left.Ref < right.Ref
	}
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	return left.Digest < right.Digest
}

func invocationEnvelopeFingerprint(envelope InvocationEnvelope) (string, error) {
	envelope.Fingerprint = ""
	data, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("invocation envelope: fingerprint: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
