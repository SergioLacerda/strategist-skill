//go:build spec

package spec_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"gopkg.in/yaml.v3"
)

// TestDojoCalibrationContractParity keeps dojo-calibration.yaml, a human-maintained
// contract, from drifting away from the code it describes: the reserved directory
// names, the record schema, the default risk policy, the fitness gate order and the
// agent's forbidden commands.
func TestDojoCalibrationContractParity(t *testing.T) {
	t.Parallel()

	raw := readFile(t, filepath.Join(repoRoot(t), "internal", "embed", "defaults", "contracts", "machine", "dojo-calibration.yaml"))
	var doc struct {
		Agent struct {
			NeverRuns []string `yaml:"never_runs"`
		} `yaml:"agent_mode"`
		Fitness struct {
			Order []string `yaml:"order"`
		} `yaml:"fitness_gates"`
		Risk struct {
			Default struct {
				FalseApproval  float64 `yaml:"max_false_approval"`
				FalseRejection float64 `yaml:"max_false_rejection"`
			} `yaml:"default"`
		} `yaml:"risk_policy"`
		Record struct {
			Schema string `yaml:"schema"`
		} `yaml:"calibration_record"`
		Storage struct {
			Reserved []string `yaml:"reserved_names"`
		} `yaml:"storage"`
	}
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("parse dojo-calibration.yaml: %v", err)
	}

	assertSameSet(t, "reserved_names", doc.Storage.Reserved, dojo.ReservedNames())
	assertSameSet(t, "fitness gate order", doc.Fitness.Order, []string{calibration.GateSeparation, calibration.GateMonotonic, calibration.GateStability})
	if doc.Record.Schema != calibration.Schema {
		t.Errorf("record schema = %q, code says %q", doc.Record.Schema, calibration.Schema)
	}
	policy := calibration.DefaultPolicy()
	if doc.Risk.Default.FalseApproval != policy.MaxFalseApproval || doc.Risk.Default.FalseRejection != policy.MaxFalseRejection {
		t.Errorf("default risk policy %+v does not match the code %+v", doc.Risk.Default, policy)
	}
	if len(doc.Agent.NeverRuns) < 3 {
		t.Errorf("the agent must have forbidden commands, got %v", doc.Agent.NeverRuns)
	}
	if need := calibration.MinSample(policy.MaxFalseApproval); need != 59 {
		t.Errorf("the contract says about 59 defect negatives for the default policy; the code needs %d", need)
	}
	if need := calibration.MinSample(policy.MaxFalseRejection); need != 19 {
		t.Errorf("the contract says about 19 golden positives for the default policy; the code needs %d", need)
	}
}
