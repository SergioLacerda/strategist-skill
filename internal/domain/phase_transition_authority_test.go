package domain

import "testing"

func TestPhaseTransitionAuthority_HappyPathOrder(t *testing.T) {
	a := NewPhaseTransitionAuthority()
	if a.Current() != PhaseBootstrap {
		t.Fatalf("initial phase = %q, want %q", a.Current(), PhaseBootstrap)
	}

	steps := []struct {
		event PhaseEvent
		want  PipelinePhase
	}{
		{EventBootstrapDone, PhaseIntake},
		{EventIntakeDone, PhaseDiscovery},
		{EventDiscoveryDone, PhaseRefinement},
	}
	for _, step := range steps {
		got, err := a.Submit(step.event)
		if err != nil {
			t.Fatalf("Submit(%q) unexpected error: %v", step.event, err)
		}
		if got != step.want {
			t.Fatalf("Submit(%q) = %q, want %q", step.event, got, step.want)
		}
	}
}

func TestPhaseTransitionAuthority_RejectsDiscoveryBeforeIntake(t *testing.T) {
	a := NewPhaseTransitionAuthority()

	_, err := a.Submit(EventDiscoveryDone)
	if err == nil {
		t.Fatal("expected an error submitting discovery-done from PhaseBootstrap")
	}
	var outOfOrder ErrOutOfOrderPhaseSubmit
	if !asErrOutOfOrderPhaseSubmit(err, &outOfOrder) {
		t.Fatalf("expected ErrOutOfOrderPhaseSubmit, got %T: %v", err, err)
	}
	if a.Current() != PhaseBootstrap {
		t.Fatalf("Current() after rejected submit = %q, want unchanged %q", a.Current(), PhaseBootstrap)
	}
}

func TestPhaseTransitionAuthority_RejectsResubmitAfterRefinement(t *testing.T) {
	a := NewPhaseTransitionAuthority()
	if _, err := a.Submit(EventBootstrapDone); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := a.Submit(EventIntakeDone); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := a.Submit(EventDiscoveryDone); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := a.Submit(EventBootstrapDone); err == nil {
		t.Fatal("expected an error submitting any event once PhaseRefinement is reached")
	}
}

func asErrOutOfOrderPhaseSubmit(err error, target *ErrOutOfOrderPhaseSubmit) bool {
	e, ok := err.(ErrOutOfOrderPhaseSubmit)
	if ok {
		*target = e
	}
	return ok
}
