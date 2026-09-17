package domain

import "testing"

func TestComposeInvocationEnvelope_ExcludesFutureAndUnselectedComponents(t *testing.T) {
	envelope, err := ComposeInvocationEnvelope(ComposeInvocationRequest{
		MissionID:       "mission-1",
		Phase:           PhaseDiscovery,
		OutputSchemaRef: "ranger-to-archivist/v1",
		Plan:            RoleInvocationPlan{Role: "ranger", Slot: "discovery", WeaponID: "brainstorming"},
		Components: []InvocationComponent{
			{Ref: "roles/ranger.yaml", Kind: "role", Phase: PhaseDiscovery, Digest: "sha256:a", Selected: true},
			{Ref: "contracts/04-refinement.md", Kind: "contract", Phase: PhaseRefinement, Digest: "sha256:b", Selected: true},
			{Ref: "unused.md", Kind: "contract", Phase: PhaseDiscovery, Digest: "sha256:c", Selected: false},
			{Ref: "protocol.md", Kind: "protocol", Digest: "sha256:d", Selected: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Components) != 2 {
		t.Fatalf("components = %+v", envelope.Components)
	}
	if envelope.RequiredContextRefs[0] != "protocol.md" || envelope.OutputSchemaRef != "ranger-to-archivist/v1" {
		t.Fatalf("unexpected envelope = %+v", envelope)
	}
}

func TestComposeInvocationEnvelope_IsDeterministicAndFingerprintChangesWithDigest(t *testing.T) {
	request := ComposeInvocationRequest{
		MissionID:       "mission-2",
		Phase:           PhaseRefinement,
		OutputSchemaRef: "archivist-to-sniper/v1",
		Plan:            RoleInvocationPlan{Role: "archivist", Slot: "refinement", WeaponID: "openspec-propose"},
		Components: []InvocationComponent{
			{Ref: "roles/archivist.yaml", Kind: "role", Phase: PhaseRefinement, Digest: "sha256:role", Selected: true},
			{Ref: "protocol.md", Kind: "protocol", Digest: "sha256:protocol", Selected: true},
		},
	}
	one, err := ComposeInvocationEnvelope(request)
	if err != nil {
		t.Fatal(err)
	}
	two, err := ComposeInvocationEnvelope(request)
	if err != nil {
		t.Fatal(err)
	}
	if one.Fingerprint != two.Fingerprint {
		t.Fatalf("same inputs produced different fingerprints: %q != %q", one.Fingerprint, two.Fingerprint)
	}
	request.Components[0].Digest = "sha256:changed"
	three, err := ComposeInvocationEnvelope(request)
	if err != nil {
		t.Fatal(err)
	}
	if one.Fingerprint == three.Fingerprint {
		t.Fatal("changed component digest did not change fingerprint")
	}
}
