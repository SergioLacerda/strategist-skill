package compile

// Whitebox tests for unexported config.go helpers whose defensive branches
// (non-map[string]any input) cannot be triggered through the public API,
// since callers always pass values of that type.

import "testing"

func TestContentByLang_NonMapInput(t *testing.T) {
	t.Parallel()
	cbl, ok := contentByLang("not a map")
	if ok || cbl != nil {
		t.Fatalf("expected (nil, false) for non-map input, got (%v, %v)", cbl, ok)
	}
}

func TestPhaseAnnouncements_NonMapInput(t *testing.T) {
	t.Parallel()
	pa, ok := phaseAnnouncements(42)
	if ok || pa != nil {
		t.Fatalf("expected (nil, false) for non-map input, got (%v, %v)", pa, ok)
	}
}
