# Strategist Skill - Templates de Implementação

Snippets prontos para aplicar as recomendações da revisão de engenharia.

---

## 1. GitHub Actions CI/CD

**Arquivo:** `.github/workflows/ci.yml`

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
    tags: ['v*']
  pull_request:
    branches: [main, develop]

env:
  GO_VERSION: "1.26.3"
  GOLANGCI_LINT_VERSION: "latest"

jobs:
  test:
    runs-on: ubuntu-latest
    name: Test & Coverage
    
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Run tests with race detector
        run: go test -race -v ./...
      
      - name: Generate coverage report
        run: |
          mkdir -p /tmp/coverage
          go test -race -coverprofile=/tmp/coverage/coverage.out -covermode=atomic ./...
      
      - name: Check coverage gate (90%)
        run: |
          fail=0
          for pkg in internal/stale internal/compile internal/install internal/embed cmd/strategist; do
            pct=$(go test -coverprofile=coverage.out -coverpkg=./$pkg/... ./$pkg/... 2>/dev/null | grep -o '[0-9.]*%' | tail -1 | tr -d '%')
            printf "%-30s %s%%\n" "$pkg" "$pct"
            ok=$(awk -v p="$pct" 'BEGIN{print (p+0 >= 90)}')
            if [ "$ok" != "1" ]; then echo "  FAIL: $pct% < 90%"; fail=1; fi
          done
          exit $fail
      
      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: /tmp/coverage/coverage.out
          flags: unittests
          name: codecov-umbrella
          fail_ci_if_error: true

  lint:
    runs-on: ubuntu-latest
    name: Lint & Security
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: ${{ env.GOLANGCI_LINT_VERSION }}
          args: --timeout=5m
      
      - name: govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
      
      - name: Trivy vulnerability scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
      
      - name: Upload Trivy results to GitHub Security tab
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'

  build:
    runs-on: ubuntu-latest
    name: Build Verification
    needs: [test, lint]
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Build binary
        run: go build -o bin/strategist ./cmd/strategist
      
      - name: Verify binary runs
        run: ./bin/strategist version

  release:
    runs-on: ubuntu-latest
    name: Release (on tag)
    needs: [test, lint, build]
    if: startsWith(github.ref, 'refs/tags/v')
    
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  # Optional: Dependabot auto-merge for patch updates
  dependabot:
    runs-on: ubuntu-latest
    if: github.actor == 'dependabot[bot]'
    permissions:
      pull-requests: write
      contents: write
    
    steps:
      - name: Enable auto-merge for Dependabot PRs
        run: gh pr merge --auto --merge "$PR_URL"
        env:
          PR_URL: ${{ github.event.pull_request.html_url }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 2. Custom Error Types

**Arquivo:** `internal/domain/errors.go`

```go
// Package domain defines core types and interfaces for Strategist.
package domain

import (
	"errors"
	"fmt"
)

// ErrorType categorizes Strategist errors for pattern matching and observability.
type ErrorType string

// Error types matching protocol.md stop conditions.
const (
	ErrorTypeSlotProviderNotFound  ErrorType = "slot_provider_not_found"
	ErrorTypeSlotRiskMismatch      ErrorType = "slot_risk_mismatch"
	ErrorTypeIntakeConflict        ErrorType = "intake_conflict_unresolved"
	ErrorTypePreflightFailed       ErrorType = "preflight_failed"
	ErrorTypeUserDeniesExecution   ErrorType = "user_denies_execution"
	ErrorTypeDiscoveryFailed       ErrorType = "discovery_failed"
	ErrorTypeRefinementFailed      ErrorType = "refinement_failed"
	ErrorTypeExecutionFailed       ErrorType = "execution_failed"
	ErrorTypeContractViolation     ErrorType = "contract_violation"
	ErrorTypeSlotWriteViolation    ErrorType = "slot_write_scope_violation"
)

// SkillError represents a structured error with type information.
// It implements standard Go error interface and can be matched with errors.Is().
type SkillError struct {
	Type       ErrorType
	Message    string
	Cause      error
	Context    map[string]any
	RetryCount int
	Transient  bool
}

// Error implements the error interface.
func (e *SkillError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap allows errors.Unwrap() to access the underlying cause.
func (e *SkillError) Unwrap() error {
	return e.Cause
}

// Is implements error matching for use with errors.Is().
// Example: errors.Is(err, &SkillError{Type: ErrorTypeSlotProviderNotFound})
func (e *SkillError) Is(target error) bool {
	t, ok := target.(*SkillError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// NewSkillError creates a new SkillError with a message.
func NewSkillError(errType ErrorType, message string) *SkillError {
	return &SkillError{
		Type:    errType,
		Message: message,
		Context: make(map[string]any),
	}
}

// WithCause wraps an underlying error.
func (e *SkillError) WithCause(cause error) *SkillError {
	e.Cause = cause
	return e
}

// WithContext adds contextual key-value pairs for observability.
func (e *SkillError) WithContext(key string, value any) *SkillError {
	e.Context[key] = value
	return e
}

// WithTransient marks the error as transient (can be retried).
func (e *SkillError) WithTransient(transient bool) *SkillError {
	e.Transient = transient
	return e
}

// IsTransient returns true if the error may be recoverable.
func (e *SkillError) IsTransient() bool {
	return e.Transient && e.RetryCount < 1
}

// CanRetry returns true if error is transient and hasn't exceeded retry limit.
func (e *SkillError) CanRetry() bool {
	return e.IsTransient() && e.RetryCount < 1
}

// IncrementRetry increments the retry counter.
func (e *SkillError) IncrementRetry() {
	e.RetryCount++
}

// Example usage in cmd:
/*
func invokeSlotProvider(slot string) error {
    provider, err := resolveProvider(slot)
    if err != nil {
        return NewSkillError(ErrorTypeSlotProviderNotFound, 
            fmt.Sprintf("cannot resolve provider for slot=%s", slot)).
            WithCause(err).
            WithContext("slot", slot).
            WithContext("resolution_paths", []string{...})
    }
    
    output, err := provider.Execute(ctx)
    if err != nil {
        // Determine if transient (network timeout) or permanent (contract violation)
        isTransient := isNetworkError(err)
        
        return NewSkillError(ErrorTypeDiscoveryFailed,
            fmt.Sprintf("provider %s failed", provider.Name())).
            WithCause(err).
            WithTransient(isTransient).
            WithContext("provider", provider.Name()).
            WithContext("phase", "discovery")
    }
    return nil
}

// Pattern matching:
if err != nil {
    var skillErr *SkillError
    if errors.As(err, &skillErr) {
        switch skillErr.Type {
        case ErrorTypeSlotProviderNotFound:
            // Handle missing provider
        case ErrorTypeDiscoveryFailed:
            if skillErr.CanRetry() {
                // Retry logic
            }
        }
    }
}
*/
```

---

## 3. Structured Logging

**Arquivo:** `internal/strategist/events.go`

```go
// Package strategist implements the mission orchestration pipeline.
package strategist

import (
	"context"
	"log/slog"
	"time"
)

// ProgressEvent represents a structured progress update.
type ProgressEvent struct {
	Phase      string         // "preflight", "intake", "discovery", etc.
	Status     string         // "running", "done", "blocked"
	Message    string         // Human-readable summary
	Details    map[string]any // Contextual data
	Timestamp  time.Time
	DurationMs int64 // If status=="done"
}

// EventLogger handles structured event emission.
type EventLogger struct {
	logger *slog.Logger
}

// NewEventLogger creates an EventLogger.
func NewEventLogger(logger *slog.Logger) *EventLogger {
	return &EventLogger{logger: logger}
}

// LogProgress emits a structured progress event.
func (el *EventLogger) LogProgress(ctx context.Context, event ProgressEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	el.logger.LogAttrs(ctx, slog.LevelInfo,
		"strategist_progress",
		slog.String("phase", event.Phase),
		slog.String("status", event.Status),
		slog.String("message", event.Message),
		slog.Int64("timestamp_unix", event.Timestamp.Unix()),
		slog.Int64("duration_ms", event.DurationMs),
		slog.Any("details", event.Details),
	)
}

// LogPhaseStart emits a phase start event.
func (el *EventLogger) LogPhaseStart(ctx context.Context, phase string, details map[string]any) {
	el.LogProgress(ctx, ProgressEvent{
		Phase:     phase,
		Status:    "running",
		Message:   phase + " phase started",
		Details:   details,
		Timestamp: time.Now(),
	})
}

// LogPhaseComplete emits a phase completion event.
func (el *EventLogger) LogPhaseComplete(ctx context.Context, phase string, durationMs int64, details map[string]any) {
	el.LogProgress(ctx, ProgressEvent{
		Phase:      phase,
		Status:     "done",
		Message:    phase + " phase completed",
		Details:    details,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
	})
}

// LogPhaseBlocked emits a phase failure event.
func (el *EventLogger) LogPhaseBlocked(ctx context.Context, phase, reason string, details map[string]any) {
	if details == nil {
		details = make(map[string]any)
	}
	details["reason"] = reason

	el.LogProgress(ctx, ProgressEvent{
		Phase:     phase,
		Status:    "blocked",
		Message:   phase + " phase failed: " + reason,
		Details:   details,
		Timestamp: time.Now(),
	})
}

// Example usage:
/*
func (s *Strategist) RunPreflight(ctx context.Context) error {
    start := time.Now()
    el := s.eventLogger
    
    el.LogPhaseStart(ctx, "preflight", map[string]any{
        "check_count": 5,
    })
    
    // ... perform preflight checks ...
    
    if err := validateSlotProviders(); err != nil {
        el.LogPhaseBlocked(ctx, "preflight", "slot_validation_failed", map[string]any{
            "error": err.Error(),
        })
        return err
    }
    
    el.LogPhaseComplete(ctx, "preflight", time.Since(start).Milliseconds(), map[string]any{
        "checks_passed": 5,
    })
    return nil
}
*/
```

---

## 4. GoReleaser Config

**Arquivo:** `.goreleaser.yml`

```yaml
version: 2

project_name: strategist
env:
  - GO111MODULE=on

before:
  hooks:
    - go mod download
    - go mod verify

builds:
  - id: strategist
    main: ./cmd/strategist
    binary: strategist
    
    goos:
      - linux
      - darwin
      - windows
    
    goarch:
      - amd64
      - arm64
    
    goarm:
      - "7"
    
    ldflags:
      - -s -w
      - -X github.com/SergioLacerda/strategist-skill/cmd/strategist.Version={{.Version}}
      - -X github.com/SergioLacerda/strategist-skill/cmd/strategist.Commit={{.Commit}}
      - -X github.com/SergioLacerda/strategist-skill/cmd/strategist.Date={{.Date}}
    
    mod_timestamp: "{{ .CommitTimestamp }}"

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    
    files:
      - README.md
      - LICENSE
      - CHANGELOG.md
      - docs/**/*
    
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
  algorithm: sha256

snapshot:
  name_template: "{{ incpatch .Version }}-snapshot"

release:
  draft: false
  prerelease: auto
  name_template: "Release {{.Version}}"
  mode: replace
  footer: |
    ## What's Changed
    
    Full changelog: https://github.com/SergioLacerda/strategist-skill/compare/{{ .PreviousTag }}...{{ .Tag }}

changelog:
  use: github
  sort: asc
  groups:
    - title: Features
      regexp: '^.*?feat(\(.+\))?!?:.+$'
      order: 0
    - title: Bug fixes
      regexp: '^.*?fix(\(.+\))?!?:.+$'
      order: 1
    - title: Deprecations
      regexp: '^.*?deprecate(\(.+\))?!?:.+$'
      order: 2
    - title: Others
      order: 999
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore'
      - Merge pull request
      - Merge branch

brews:
  - name: strategist
    repository:
      owner: SergioLacerda
      name: homebrew-strategist
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    
    directory: Formula
    homepage: "https://github.com/SergioLacerda/strategist-skill"
    description: "Multi-phase mission orchestrator for AI agents"
    license: "MIT"
    
    test: |
      system "#{bin}/strategist version"

dockers:
  - image_templates:
      - "ghcr.io/SergioLacerda/strategist:{{ .Version }}"
      - "ghcr.io/SergioLacerda/strategist:latest"
    
    dockerfile: Dockerfile
    build_flag_templates:
      - "--pull"
      - "--label=org.opencontainers.image.created={{.Date}}"
      - "--label=org.opencontainers.image.title={{.ProjectName}}"
      - "--label=org.opencontainers.image.version={{.Version}}"
      - "--label=org.opencontainers.image.revision={{.FullCommit}}"

signs:
  - cmd: cosign
    signature: "${artifact}.sig"
    certificate: "${artifact}.pem"
    args:
      - sign-blob
      - --oidc-provider=github
      - "--output-signature=${signature}"
      - "--output-certificate=${certificate}"
      - "${artifact}"
    artifacts: checksum
    output: true
```

---

## 5. Validation Command

**Arquivo:** `cmd/strategist/validate.go`

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all configuration files",
	Long: `Validate Strategist configuration files for correctness and completeness.
	
Checks performed:
  - active.yaml structure and required fields
  - Persona files (pragmatic.yaml, epic.yaml)
  - Roles configuration
  - Knowledge index references
  - Slot provider contracts`,
	RunE: func(_ *cobra.Command, _ []string) error {
		root := ".strategist"
		var errs []error

		// Validate active.yaml
		if err := validateActiveYAML(filepath.Join(root, "active.yaml")); err != nil {
			errs = append(errs, fmt.Errorf("active.yaml: %w", err))
		}

		// Validate personas
		for _, mode := range []string{"pragmatic", "epic"} {
			path := filepath.Join(root, "personas", mode+".yaml")
			if err := validatePersonaYAML(path); err != nil {
				errs = append(errs, fmt.Errorf("personas/%s.yaml: %w", mode, err))
			}
		}

		// Validate roles
		rolesDir := filepath.Join(root, "roles")
		entries, err := os.ReadDir(rolesDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if !entry.Name()[len(entry.Name())-5:] == ".yaml" {
					continue
				}
				path := filepath.Join(rolesDir, entry.Name())
				if err := validateRolesYAML(path); err != nil {
					errs = append(errs, fmt.Errorf("roles/%s: %w", entry.Name(), err))
				}
			}
		}

		// Validate knowledge index
		if err := validateKnowledgeIndex(filepath.Join(root, "knowledge.index.yaml")); err != nil {
			errs = append(errs, fmt.Errorf("knowledge.index.yaml: %w", err))
		}

		// Validate schemas
		schemasDir := filepath.Join(root, "schemas")
		if _, err := os.Stat(schemasDir); err == nil {
			entries, _ := os.ReadDir(schemasDir)
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				path := filepath.Join(schemasDir, entry.Name())
				if err := validateSchemaYAML(path); err != nil {
					errs = append(errs, fmt.Errorf("schemas/%s: %w", entry.Name(), err))
				}
			}
		}

		if len(errs) > 0 {
			fmt.Fprintln(os.Stderr, "❌ Validation failed with errors:")
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  • %v\n", e)
			}
			return fmt.Errorf("validation failed: %d error(s)", len(errs))
		}

		fmt.Println("✅ All configurations valid")
		return nil
	},
}

func validateActiveYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found")
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Required fields
	required := []string{"mode", "base_path", "roles_config"}
	for _, field := range required {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate mode
	if mode, ok := config["mode"].(string); ok {
		if mode != "pragmatic" && mode != "epic" {
			return fmt.Errorf("invalid mode: %s (must be 'pragmatic' or 'epic')", mode)
		}
	}

	return nil
}

func validatePersonaYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found")
		}
		return fmt.Errorf("read failed: %w", err)
	}

	var persona map[string]any
	if err := yaml.Unmarshal(data, &persona); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Required: tone_directive, phase_labels
	if _, ok := persona["tone_directive"]; !ok {
		return fmt.Errorf("missing tone_directive")
	}
	if _, ok := persona["phase_labels"]; !ok {
		return fmt.Errorf("missing phase_labels")
	}

	return nil
}

func validateRolesYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	var roles map[string]any
	if err := yaml.Unmarshal(data, &roles); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Required: slots with discovery, refinement, execution
	slots, ok := roles["slots"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing 'slots' key")
	}

	required := []string{"discovery", "refinement", "execution"}
	for _, slot := range required {
		if _, ok := slots[slot]; !ok {
			return fmt.Errorf("missing slot: %s", slot)
		}
	}

	return nil
}

func validateKnowledgeIndex(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Optional file
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat failed: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	var index map[string]any
	if err := yaml.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	return nil
}

func validateSchemaYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	var schema map[string]any
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
```

---

## 6. Benchmark Test

**Arquivo:** `internal/stale/stale_bench_test.go`

```go
package stale

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCheck(b *testing.B) {
	tmpDir := b.TempDir()
	
	// Setup test data
	sourcePath := filepath.Join(tmpDir, "source.gz")
	destPath := filepath.Join(tmpDir, "dest")
	
	// Create dummy files
	os.WriteFile(sourcePath, []byte("test"), 0o644)
	os.WriteFile(destPath, []byte("test"), 0o644)
	
	checker := &Checker{
		SourcePath:      sourcePath,
		DestinationPath: destPath,
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		checker.Check()
	}
}

func BenchmarkCheck_MissingDestination(b *testing.B) {
	tmpDir := b.TempDir()
	sourcePath := filepath.Join(tmpDir, "source.gz")
	destPath := filepath.Join(tmpDir, "nonexistent")
	
	os.WriteFile(sourcePath, []byte("test"), 0o644)
	
	checker := &Checker{
		SourcePath:      sourcePath,
		DestinationPath: destPath,
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		checker.Check()
	}
}
```

Run com: `go test -bench=. -benchmem ./internal/stale`

---

## 7. Makefile Aprimorado

```makefile
.PHONY: help build test test-verbose lint fmt security cover cover-gate cover-html validate clean install-local release

# Default target
.DEFAULT_GOAL := help

GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)
GO := go

help: ## Show this help message
	@echo "Strategist Skill — Makefile targets"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Build targets
build: ## Build the strategist CLI binary
	$(GO) build -o bin/strategist ./cmd/strategist
	@echo "✓ Built: bin/strategist"

build-all: clean ## Build for all platforms
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/strategist-linux-amd64 ./cmd/strategist
	GOOS=darwin GOARCH=amd64 $(GO) build -o bin/strategist-darwin-amd64 ./cmd/strategist
	GOOS=darwin GOARCH=arm64 $(GO) build -o bin/strategist-darwin-arm64 ./cmd/strategist
	GOOS=windows GOARCH=amd64 $(GO) build -o bin/strategist-windows-amd64.exe ./cmd/strategist
	@echo "✓ Built all binaries in bin/"

# Test targets
test: ## Run all tests with race detector
	$(GO) test -race ./...

test-verbose: ## Run tests with verbose output
	$(GO) test -race -v ./...

test-count: ## Run tests multiple times (race condition detection)
	$(GO) test -race -count=5 ./...

# Linting targets
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./... --deadline=5m

fmt: ## Format code with gofmt
	@echo "Running gofmt..."
	@gofmt -s -w .
	@echo "✓ Code formatted"

fmt-check: ## Check if code is properly formatted
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "❌ Code not formatted:"; \
		gofmt -s -l .; \
		exit 1; \
	else \
		echo "✓ Code properly formatted"; \
	fi

vet: ## Run go vet
	$(GO) vet ./...

# Security targets
security: ## Check for vulnerabilities
	govulncheck ./...

security-full: ## Comprehensive security check
	@echo "Running security checks..."
	$(GOLANGCI_LINT) run ./... \
		--enable=gosec \
		--enable=errorlint \
		--enable=wastedassign \
		--deadline=5m
	govulncheck ./...
	@echo "✓ Security checks passed"

# Coverage targets
cover: ## Show per-package coverage percentages
	@echo "Coverage by package:"
	@for pkg in internal/stale internal/compile internal/install internal/embed cmd/strategist; do \
		pct=$$($(GO) test -coverprofile=/tmp/coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null | grep -o '[0-9.]*%' | tail -1); \
		printf "  %-30s %s\n" "$$pkg" "$$pct"; \
	done

cover-gate: ## Verify all packages meet 90% coverage minimum
	@echo "Checking coverage gate (90% minimum)..."
	@fail=0; \
	for pkg in internal/stale internal/compile internal/install internal/embed cmd/strategist; do \
		pct=$$($(GO) test -coverprofile=/tmp/coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null | grep -o '[0-9.]*%' | tail -1 | tr -d '%'); \
		printf "  %-30s %s%%\n" "$$pkg" "$$pct"; \
		ok=$$(awk -v p="$$pct" 'BEGIN{print (p+0 >= 90)}'); \
		if [ "$$ok" != "1" ]; then echo "    FAIL: $$pct% < 90%"; fail=1; fi; \
	done; \
	exit $$fail

cover-html: ## Generate HTML coverage report
	$(GO) test -race -coverprofile=/tmp/coverage.out -coverpkg=./internal/... ./internal/... ./tests/...
	$(GO) tool cover -html=/tmp/coverage.out

# Validation targets
validate: ## Validate configuration (run 'make install-local' first)
	@if ! command -v strategist &> /dev/null; then \
		echo "strategist not in PATH. Run 'make install-local' first."; \
		exit 1; \
	fi
	strategist validate

# Dependency targets
deps: ## Check and update dependencies
	$(GO) get -u ./...
	$(GO) mod tidy

deps-check: ## Check for unused dependencies
	@$(GO) mod tidy -v

# Installation targets
install-local: build ## Install binary to ~/.local/bin/
	@mkdir -p ~/.local/bin
	install -m 755 bin/strategist ~/.local/bin/strategist
	@echo "✓ Installed: ~/.local/bin/strategist"

# Release targets
release: ## Create a release (requires git tags)
	goreleaser release --clean

release-snapshot: ## Create a snapshot release (no upload)
	goreleaser release --snapshot --rm-dist

# Cleanup targets
clean: ## Remove build artifacts
	rm -rf bin/ dist/ coverage.out /tmp/coverage.out
	@echo "✓ Cleaned"

distclean: clean ## Remove all generated files
	rm -rf .strategist/ .compiled/
	@echo "✓ Fully cleaned"

# CI/local integration
ci: fmt-check lint security cover-gate test ## Run all CI checks locally
	@echo ""
	@echo "✅ All CI checks passed!"

# Development targets
dev: build test lint fmt ## Build, test, lint, and format
	@echo ""
	@echo "✅ Dev build complete!"

# Watch mode (requires 'watchexec')
watch: ## Watch for changes and re-run tests
	@command -v watchexec >/dev/null || { echo "watchexec required: cargo install watchexec"; exit 1; }
	watchexec -r '$(GO) test -race ./...'

# Version target
version: ## Show version information
	@echo "Strategist Skill v1.0.0"
	@echo "Go: $$(go version | awk '{print $$3}')"
	@echo "Platform: $$(go env GOOS)/$$(go env GOARCH)"
```

---

Estes templates estão prontos para implementação. Comece pelos arquivos CI/CD e Custom Error Types para máximo impacto.
