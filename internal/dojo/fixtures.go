package dojo

import (
	"fmt"
	"os"
	"path/filepath"
)

// SyntheticTopics make the synthetic fixture packages differ in content while
// each stays well formed and internally consistent.
var SyntheticTopics = []string{
	"audit-log", "retry-policy", "cache-warmup", "feature-flags", "rate-limiter", "report-export", "health-probe", "schema-lint",
	"queue-drain", "token-rotation", "config-reload", "metrics-rollup", "backup-verify", "index-rebuild", "session-expiry", "webhook-replay",
	"quota-check", "locale-fallback", "log-redaction", "circuit-breaker", "batch-import", "event-dedupe", "lease-renewal", "cron-guard",
	"snapshot-diff", "secret-scan", "trace-sampling", "job-priority", "tenant-isolation", "schema-migrate", "notify-digest", "access-review",
}

// WriteSyntheticPackages writes n well-formed synthetic packages under dir
// (one directory each, the four canonical files). They contain nothing of any
// mission, so a calibration over them sends no real content. n is capped at the
// number of topics.
func WriteSyntheticPackages(dir string, n int) ([]string, error) {
	n = min(max(n, 1), len(SyntheticTopics))
	names := make([]string, 0, n)
	for _, topic := range SyntheticTopics[:n] {
		name := "synthetic-" + topic
		if err := writeSyntheticPackage(filepath.Join(dir, name), topic); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func writeSyntheticPackage(dir, topic string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain
		return fmt.Errorf("dojo: create %s: %w", dir, err)
	}
	files := map[string]string{
		"tasks.md":    syntheticTasks(topic),
		"analysis.md": fmt.Sprintf("---\nmission_status: documentation_applied\n---\n\n## affected_scope\n- allowed: docs/dojo/%[1]s/**, internal/%[1]s/**, cmd/strategist/%[1]s.go\n- forbidden: internal/domain/**, .strategist/**\n", topic),
		"proposal.md": "## Why\nsynthetic fixture\n",
		"design.md":   "## Context\nsynthetic fixture\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
			return fmt.Errorf("dojo: write %s: %w", name, err)
		}
	}
	return nil
}

func syntheticTasks(topic string) string {
	return fmt.Sprintf(`---
approved_scope: |
  - allowed: docs/dojo/%[1]s/**, internal/%[1]s/**, cmd/strategist/%[1]s.go
  - forbidden: internal/domain/**, .strategist/**
acceptance_checks:
  - "`+"`go test ./internal/%[1]s/...`"+` exits 0"
  - "docs/dojo/%[1]s/README.md contains the section Usage"
---
- [ ] 1.1 [analysis_artifact] Record the author's answers about %[1]s in docs/dojo/%[1]s/design.md; verify the file lists every answer with its date.
- [ ] 1.2 [implementation_handoff] Specify the %[1]s configuration file and its defaults in docs/dojo/%[1]s/config.md; verify the file holds a table of keys and defaults.
- [ ] 2.1 [implementation_handoff] Build the %[1]s package under internal/%[1]s; verify its tests cover the empty, single and many cases.
- [ ] 2.2 [implementation_handoff] Wire %[1]s into the command layer in cmd/strategist/%[1]s.go. The wiring MUST NOT change any existing flag; verify `+"`go test ./cmd/strategist/...`"+` exits 0.
- [ ] 3.1 [analysis_artifact] Report what %[1]s measured in docs/dojo/%[1]s/README.md; verify the report lists measured values only.
`, topic)
}
