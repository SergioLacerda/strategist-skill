package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMissionStatusBranches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status MissionEngineStatus
		want   string
	}{
		{"no id", MissionEngineStatus{}, "mission id"},
		{"no phase", MissionEngineStatus{MissionID: "m"}, "phase and state"},
		{"neg attempt", MissionEngineStatus{MissionID: "m", Phase: PhaseIntake, State: StateInit, HandoffAttempt: -1}, "negative"},
		{"meta no attempt", MissionEngineStatus{MissionID: "m", Phase: PhaseIntake, State: StateInit, HandoffStatus: "x"}, "positive attempt"},
		{"blocked phase", MissionEngineStatus{MissionID: "m", Phase: PhaseBlocked, State: StateInit}, "blocked phase"},
		{"init late", MissionEngineStatus{MissionID: "m", Phase: PhaseExecution, State: StateInit}, "early pipeline"},
		{"bad state", MissionEngineStatus{MissionID: "m", Phase: PhaseDone, State: StateExecution}, "invalid for phase"},
	}
	for _, tc := range cases {
		mustFail(t, validateMissionStatus(tc.status), tc.want)
	}
	if err := validateMissionStatus(MissionEngineStatus{MissionID: "m", Phase: PhaseIntake, State: StateInit}); err != nil {
		t.Fatal(err)
	}
}

func TestMaterializeContextRequestAndReferenceValidation(t *testing.T) {
	t.Parallel()
	_, err := MaterializeContext("", nil, 0, 0)
	mustFail(t, err, "root is required")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"", "/abs", "a/../a.md", ".", "..", "../x"} {
		_, err := MaterializeContext(root, []ContextReference{{Ref: ref, Kind: "k"}}, 0, 0)
		mustFail(t, err, "invalid relative reference")
	}
	_, err = MaterializeContext(root, []ContextReference{{Ref: "a.md", Kind: "k"}, {Ref: "b.md", Kind: "k"}}, 1, 0)
	mustFail(t, err, "reference limit")
	_, err = MaterializeContext(root, []ContextReference{{Ref: "missing.md", Kind: "k"}}, 0, 0)
	mustFail(t, err, "blocked reference")
	_, err = MaterializeContext(root, []ContextReference{{Ref: "a.md", Kind: "k"}}, 0, 2)
	mustFail(t, err, "byte limit")
	_, err = MaterializeContext(root, []ContextReference{{Ref: "a.md", Kind: "k", Digest: "sha256:bad"}}, 0, 0)
	mustFail(t, err, "digest mismatch")
	_, err = MaterializeContext(root, []ContextReference{{Ref: "a.md", Kind: "k"}, {Ref: "a.md", Kind: "k"}}, 0, 0)
	mustFail(t, err, "duplicate")
	res, err := MaterializeContext(root, []ContextReference{{Ref: "a.md", Kind: "k"}, {Ref: "a.md", Kind: "j"}}, 0, 0)
	if err != nil || len(res.References) != 2 || res.References[0].Kind != "j" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !containsParentPathComponent("x/../y") || containsParentPathComponent("x/y") {
		t.Fatal("containsParentPathComponent")
	}
}

func TestInvocationEnvelopeValidationAndOrdering(t *testing.T) {
	t.Parallel()
	plan := RoleInvocationPlan{Role: "r", Slot: "discovery", WeaponID: "w"}
	cases := []struct {
		req  ComposeInvocationRequest
		want string
	}{
		{ComposeInvocationRequest{Phase: PhaseDiscovery, Plan: plan}, "mission id"},
		{ComposeInvocationRequest{MissionID: "m", Plan: plan}, "phase is required"},
		{ComposeInvocationRequest{MissionID: "m", Phase: PhaseDiscovery}, "role, slot, and provider"},
		{ComposeInvocationRequest{MissionID: "m", Phase: PhaseDiscovery, Plan: plan}, "output schema"},
	}
	for _, tc := range cases {
		_, err := ComposeInvocationEnvelope(tc.req)
		mustFail(t, err, tc.want)
	}
	comp := func(ref, kind, digest string) InvocationComponent {
		return InvocationComponent{Ref: ref, Kind: kind, Digest: digest, Selected: true}
	}
	env, err := ComposeInvocationEnvelope(ComposeInvocationRequest{
		MissionID: "m", Phase: PhaseDiscovery, Plan: plan, OutputSchemaRef: "s",
		Components: []InvocationComponent{comp("b", "k", "1"), comp("a", "z", "1"), comp("a", "k", "2"), comp("a", "k", "1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := env.Components
	if got[0].Kind != "k" || got[0].Digest != "1" || got[1].Digest != "2" || got[2].Kind != "z" || got[3].Ref != "b" || env.Fingerprint == "" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestIsKnownPluginPermission(t *testing.T) {
	t.Parallel()
	if !IsKnownPluginPermission(PluginPermissionReadWorkspace) || IsKnownPluginPermission("nope") {
		t.Fatal("permission vocabulary")
	}
	err := AdapterContract{RequestedPermissions: []PluginPermission{"nope"}, SupportedSlots: []string{"bogus"}}.Validate()
	mustFail(t, err, "invalid permission")
	mustFail(t, err, "invalid slot")
}
