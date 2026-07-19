package compile_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
)

func TestActiveConfig_Validate_Valid(t *testing.T) {
	t.Parallel()
	cfg := compile.ActiveConfig{
		Mode:     "epic",
		BasePath: ".analysis",
		Slots:    map[string]string{"discovery": "brainstorming"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error for valid config: %v", err)
	}
}

func TestActiveConfig_Validate_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  compile.ActiveConfig
		want string
	}{
		{
			name: "missing mode",
			cfg:  compile.ActiveConfig{BasePath: ".analysis", Slots: map[string]string{"x": "y"}},
			want: "mode is required",
		},
		{
			name: "missing base_path",
			cfg:  compile.ActiveConfig{Mode: "epic", Slots: map[string]string{"x": "y"}},
			want: "base_path is required",
		},
		{
			name: "empty slots",
			cfg:  compile.ActiveConfig{Mode: "epic", BasePath: ".analysis"},
			want: "slots must have at least one entry",
		},
		{
			name: "all missing",
			cfg:  compile.ActiveConfig{},
			want: "mode is required",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.Validate()
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if msg := err.Error(); len(msg) == 0 {
				t.Fatalf("error message is empty for %s", tc.name)
			}
		})
	}
}

func TestPersonaConfig_Validate_Valid(t *testing.T) {
	t.Parallel()
	p := compile.PersonaConfig{
		ID:            "epic",
		ToneDirective: "be precise",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error for valid persona: %v", err)
	}
}

func TestPersonaConfig_Validate_MissingFields(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		p    compile.PersonaConfig
	}{
		{"missing id", compile.PersonaConfig{ToneDirective: "be precise"}},
		{"missing tone_directive", compile.PersonaConfig{ID: "epic"}},
		{"all missing", compile.PersonaConfig{}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.p.Validate(); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
