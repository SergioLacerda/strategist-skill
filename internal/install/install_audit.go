package install

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"gopkg.in/yaml.v3"
)

// installAuditRelPath is the durable ledger of install and upgrade runs. It
// exists so a configuration reset can be attributed: before it, the only
// trace of a forced install was a stderr warning.
const installAuditRelPath = "memory/install-events.jsonl"

// AuditSchema identifies the record shape.
const AuditSchema = "strategist-install-audit/v1"

// AuditRecord is one install or upgrade run. It carries flags and slot
// provider ids only: no file contents, secrets, provider keys or paths.
type AuditRecord struct {
	Schema        string            `json:"schema"`
	At            string            `json:"at"`
	Command       string            `json:"command"`
	Mode          string            `json:"mode,omitempty"`
	Force         bool              `json:"force"`
	ResetConfig   bool              `json:"reset_config"`
	SlotsBefore   map[string]string `json:"slots_before,omitempty"`
	SlotsAfter    map[string]string `json:"slots_after,omitempty"`
	ConfigChanged bool              `json:"config_changed"`
}

// readSlotBindings returns the slots block of active.yaml, or nil when the file
// is absent or unreadable (an audit never fails an install).
func readSlotBindings(strategistDir string) map[string]string {
	raw, err := os.ReadFile(filepath.Join(strategistDir, names.ActiveYAMLName)) //nolint:gosec // fixed file under the runtime dir
	if err != nil {
		return nil
	}
	var cfg struct {
		Slots map[string]string `yaml:"slots"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil
	}
	return cfg.Slots
}

func installMode(cfg domain.InstallConfig) string {
	if cfg.Wizard {
		return "wizard"
	}
	return "silent"
}

// recordInstallAudit appends the audit record of a finished install. It is
// best-effort: a failure is logged and never fails the install.
func recordInstallAudit(strategistDir string, cfg domain.InstallConfig, before map[string]string) {
	after := readSlotBindings(strategistDir)
	appendAuditRecord(strategistDir, AuditRecord{
		Command: "install", Mode: installMode(cfg), Force: cfg.Force, ResetConfig: cfg.ResetConfig,
		SlotsBefore: before, SlotsAfter: after, ConfigChanged: !reflect.DeepEqual(before, after),
	})
}

// recordUpgradeAudit appends the audit record of a finished upgrade.
func recordUpgradeAudit(strategistDir string, force bool, before map[string]string) {
	after := readSlotBindings(strategistDir)
	appendAuditRecord(strategistDir, AuditRecord{
		Command: "upgrade", Force: force,
		SlotsBefore: before, SlotsAfter: after, ConfigChanged: !reflect.DeepEqual(before, after),
	})
}

func appendAuditRecord(strategistDir string, rec AuditRecord) {
	rec.Schema = AuditSchema
	rec.At = time.Now().UTC().Format(time.RFC3339)
	if err := writeAuditLine(filepath.Join(strategistDir, filepath.FromSlash(installAuditRelPath)), rec); err != nil {
		slog.Warn("[Strategist] install audit record not written",
			telemetry.AttrComponent, "install", "reason", err.Error())
	}
}

func writeAuditLine(path string, rec AuditRecord) (err error) {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal install audit record: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create install audit dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644) //nolint:gosec // G304: ledger path is owned by the runtime memory domain
	if err != nil {
		return fmt.Errorf("open install audit ledger: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close install audit ledger: %w", closeErr)
		}
	}()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write install audit record: %w", err)
	}
	return nil
}
