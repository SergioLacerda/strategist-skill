package telemetry

import "testing"

func TestSanitizePath_absolute(t *testing.T) {
	got := SanitizePath("/home/user/projects/myapp")
	if got != "<redacted-path>" {
		t.Errorf("expected <redacted-path>, got %q", got)
	}
}

func TestSanitizePath_relative(t *testing.T) {
	got := SanitizePath(".analysis/refined")
	if got != ".analysis/refined" {
		t.Errorf("expected path unchanged, got %q", got)
	}
}

func TestSanitizePath_empty(t *testing.T) {
	got := SanitizePath("")
	if got != "" {
		t.Errorf("expected empty string unchanged, got %q", got)
	}
}
