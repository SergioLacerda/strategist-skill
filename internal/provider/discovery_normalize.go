package provider

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

func validateDiscoveryRequest(request DiscoveryWeaponRequest) error {
	if strings.TrimSpace(request.MissionID) == "" {
		return fmt.Errorf("mission id is required")
	}
	if request.Role != "ranger" {
		return fmt.Errorf("discovery Weapon role must be ranger, got %q", request.Role)
	}
	if request.Slot != string(domain.SlotDiscovery) {
		return fmt.Errorf("discovery Weapon slot must be discovery, got %q", request.Slot)
	}
	if strings.TrimSpace(request.ProviderID) == "" {
		return fmt.Errorf("provider id is required")
	}
	if strings.TrimSpace(request.ArtifactPath) == "" {
		return fmt.Errorf("artifact path is required")
	}
	if err := domain.ValidateSlotWrite(domain.SlotWriteScope{
		SlotName:      "discovery",
		AllowedPrefix: ".analysis/pending/",
		AllowedExt:    ".md",
	}, request.ArtifactPath); err != nil {
		return fmt.Errorf("validate discovery artifact path: %w", err)
	}
	return nil
}

func normalizeDiscoveryArtifact(request DiscoveryWeaponRequest, response DiscoveryWeaponResponse) ([]byte, error) {
	if len(response.Artifact) == 0 {
		return nil, fmt.Errorf("weapon returned an empty discovery artifact")
	}
	if len(response.Artifact) > maxDiscoveryArtifactBytes {
		return nil, fmt.Errorf("discovery artifact exceeds %d bytes", maxDiscoveryArtifactBytes)
	}
	if !utf8.Valid(response.Artifact) {
		return nil, fmt.Errorf("discovery artifact is not valid UTF-8")
	}

	frontmatter, body, err := splitDiscoveryFrontmatter(response.Artifact)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("discovery artifact body is empty")
	}
	// Provider-controlled identity fields are overwritten by Ranger's trusted
	// request and host evidence before the artifact reaches the pending path.
	if _, ok := frontmatter["sources_consulted"].([]any); !ok {
		// The handoff requires the list; an absent one means "no source reported".
		frontmatter["sources_consulted"] = []any{}
	}
	frontmatter["schema_version"] = DiscoveryArtifactSchemaVersion
	frontmatter["mission_id"] = request.MissionID
	frontmatter["mission_status"] = "ranger_pending"
	frontmatter["analysis_artifact_path"] = request.ArtifactPath
	frontmatter["provider_id"] = request.ProviderID
	frontmatter["invocation_evidence"] = strings.TrimSpace(response.InvocationEvidence)

	encoded, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized discovery metadata: %w", err)
	}
	return append(append([]byte("---\n"), append(encoded, []byte("---\n\n")...)...), append(bytes.TrimSpace(body), '\n')...), nil
}

func splitDiscoveryFrontmatter(raw []byte) (map[string]any, []byte, error) {
	content := bytes.TrimSpace(raw)
	frontmatter := map[string]any{}
	if !bytes.HasPrefix(content, []byte("---")) {
		return frontmatter, content, nil
	}
	lineEnd := bytes.IndexByte(content, '\n')
	if lineEnd < 0 || strings.TrimSpace(string(content[:lineEnd])) != "---" {
		return nil, nil, fmt.Errorf("discovery artifact frontmatter is malformed")
	}
	closing := bytes.Index(content[lineEnd+1:], []byte("\n---"))
	if closing < 0 {
		return nil, nil, fmt.Errorf("discovery artifact frontmatter is unclosed")
	}
	closing += lineEnd + 1
	if err := yaml.Unmarshal(content[lineEnd+1:closing], &frontmatter); err != nil {
		return nil, nil, fmt.Errorf("parse discovery artifact frontmatter: %w", err)
	}
	return frontmatter, content[closing+4:], nil
}

func invocationFailure(err error) error {
	return fmt.Errorf("role_invocation_failed: %w", err)
}

// DiscoveryOutputContract tells a Weapon executor the shape Ranger's
// normalization and the Archivist handoff require. It is the single source for
// both the host-bridge prompt and the manual `mission invoke` request.
const DiscoveryOutputContract = "Return Markdown that starts with YAML frontmatter containing a `sources_consulted` list " +
	"(each item: source_path, content_fingerprint, coverage_status; use an empty list when no source was opened) " +
	"and optionally `confidence_score`, `discovery_subtype` and `evaluation_verdict`. " +
	"The body must contain these exact second-level headings: `## mission_objective`, `## known_facts`, " +
	"`## uncertainties`, `## affected_scope`, `## side_quests`, `## confidence_summary`, " +
	"`## recommended_refinement_focus` and `## handoff`. Strategist overwrites identity fields " +
	"(mission_id, mission_status, schema_version, provider_id)."

// DiscoveryExecutionContract adapts an interactive external skill to Ranger's
// single-shot, read-only Weapon boundary. The host executes the compiled
// payload, but Strategist remains the owner of user interaction and approval.
const DiscoveryExecutionContract = "Execute the compiled Weapon payload once as a single-shot, read-only Ranger discovery call. " +
	"Use the payload's analysis methods, but do not ask the user questions, wait for an approval, invoke another skill, " +
	"write files, or perform implementation. Any interactive or implementation gate inside the payload belongs to its " +
	"standalone workflow and is not part of this adapter. Return only the discovery handoff required by output_contract."
