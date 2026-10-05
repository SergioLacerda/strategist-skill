package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func calibrationBasePath(root string) string { return filepath.Join(filepath.Dir(root), ".analysis") }

func publish(t *testing.T, root string, threshold float64, mutate func(*calibration.Meta, *calibration.Report)) calibration.Record {
	t.Helper()
	meta := calibration.Meta{Provider: "jev", Model: "jev-1.13.0", Profile: handoffconsumer.ProfileID, ProfileVersion: handoffconsumer.ProfileVersion(calibration.BoundaryPackage),
		CorpusDigest: "sha256:c", CatalogVersion: calibration.CatalogVersion(), Seed: 1, Now: time.Now()}
	held := calibration.SetResult{Threshold: threshold, Positives: 20, Defects: 70}
	report := calibration.Report{Status: calibration.StatusRecommended, Threshold: &threshold, HeldOut: &held, Policy: calibration.DefaultPolicy(),
		Counts: calibration.Counts{Golden: 60, Defects: 300}, Gates: calibration.FitnessGates(nil, calibration.DefaultLimits())}
	if mutate != nil {
		mutate(&meta, &report)
	}
	record, err := calibration.NewRecord(report, meta)
	require.NoError(t, err)
	require.NoError(t, dojo.SaveCalibration(calibrationBasePath(root), record))
	return record
}

func calibrationRig(t *testing.T) string {
	t.Helper()
	root := bareRoot(t)
	enableIntegration(t, root, jevtest.Start(t, jevtest.Reply([]byte(probeBody))), nil)
	return root
}

func runPin(t *testing.T, root, id string, confirm bool) (string, error) {
	t.Helper()
	return runCalibrationCommand(t, func(c *cobraCmd) error {
		return runCalibrationPin(c, calibrationOptions{integrationsOptions: integrationsOptions{Root: root}, confirm: confirm}, id)
	})
}

func pinnedID(t *testing.T, root string) string {
	t.Helper()
	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	return file.Providers["jev"].CalibrationID
}

func TestPinShowsTheNumbersFirstAndWritesOnlyWithConfirm(t *testing.T) {
	root := calibrationRig(t)
	record := publish(t, root, 0.84, nil)

	out, err := runPin(t, root, record.ID, false)
	require.NoError(t, err)
	assert.Contains(t, out, "risk policy: false approval <= 5.0%")
	assert.Contains(t, out, "recommended threshold: 0.84")
	assert.Contains(t, out, "not pinned")
	assert.Empty(t, pinnedID(t, root))

	out, err = runPin(t, root, record.ID, true)
	require.NoError(t, err)
	assert.Contains(t, out, "pinned "+record.ID)
	assert.Equal(t, record.ID, pinnedID(t, root))
}

func TestPinRefusesWhatTheHandoffCouldNotUse(t *testing.T) {
	root := calibrationRig(t)
	notFit := publish(t, root, 0.9, func(_ *calibration.Meta, r *calibration.Report) {
		r.Status, r.Threshold, r.HeldOut, r.FailingGate = calibration.StatusNotFit, nil, nil, "separation"
	})
	otherModel := publish(t, root, 0.9, func(m *calibration.Meta, _ *calibration.Report) { m.Model = "jev-2.0.0"; m.Seed = 2 })
	oldProfile := publish(t, root, 0.9, func(m *calibration.Meta, _ *calibration.Report) { m.ProfileVersion = "00000000"; m.Seed = 3 })
	for name, id := range map[string]string{"not recommended": notFit.ID, "another model": otherModel.ID, "old profile": oldProfile.ID, "latest": "latest", "unknown": "cal-20200101-deadbeef"} {
		_, err := runPin(t, root, id, true)
		require.Error(t, err, name)
		assert.Empty(t, pinnedID(t, root), name)
	}
}

func TestPinNeedsADeclaredIntegration(t *testing.T) {
	root := bareRoot(t)
	record := publish(t, root, 0.84, nil)
	_, err := runPin(t, root, record.ID, true)
	require.ErrorContains(t, err, "integrations enable")
}

func TestDoctorSaysCalibratedOnlyForAValidPinnedCalibration(t *testing.T) {
	root := calibrationRig(t)
	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "calibration: não calibrado")

	record := publish(t, root, 0.84, nil)
	_, err = runPin(t, root, record.ID, true)
	require.NoError(t, err)
	out, err = runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "calibration: calibrado ("+record.ID)
	assert.Contains(t, out, "threshold 0.84")

	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	provider := file.Providers["jev"]
	provider.Model, provider.AllowModelAlias = "jev-1.14.0", true
	file.Providers["jev"] = provider
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), file))
	out, err = runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "calibration: não calibrado (model_changed")
}

func TestDoctorReportsAMissingPinnedRecordAsNotCalibrated(t *testing.T) {
	root := calibrationRig(t)
	record := publish(t, root, 0.84, nil)
	_, err := runPin(t, root, record.ID, true)
	require.NoError(t, err)
	require.NoError(t, removeCalibration(calibrationBasePath(root), record.ID))
	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "não calibrado (record_unavailable")
}

func TestUnpinGoesBackToTheConfiguredThreshold(t *testing.T) {
	root := calibrationRig(t)
	record := publish(t, root, 0.84, nil)
	_, err := runPin(t, root, record.ID, true)
	require.NoError(t, err)
	out, err := runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationUnpin(c, integrationsOptions{Root: root}) })
	require.NoError(t, err)
	assert.Contains(t, out, "unpinned")
	assert.Empty(t, pinnedID(t, root))
	out, err = runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationUnpin(c, integrationsOptions{Root: root}) })
	require.NoError(t, err)
	assert.Contains(t, out, "no calibration is pinned")
}

func TestListAndShow(t *testing.T) {
	root := calibrationRig(t)
	out, err := runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationList(c, integrationsOptions{Root: root}) })
	require.NoError(t, err)
	assert.Contains(t, out, "no calibration published yet")

	record := publish(t, root, 0.84, nil)
	out, err = runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationList(c, integrationsOptions{Root: root}) })
	require.NoError(t, err)
	assert.Contains(t, out, record.ID)
	assert.Contains(t, out, "recommended")

	out, err = runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationShow(c, integrationsOptions{Root: root}, "") })
	require.NoError(t, err)
	assert.Contains(t, out, "no calibration is pinned")
	out, err = runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationShow(c, integrationsOptions{Root: root}, record.ID) })
	require.NoError(t, err)
	assert.Contains(t, out, "sample: 60 golden positives, 300 defect negatives")
	assert.Contains(t, out, "judge: auc")
}

func TestTheConfigRejectsAFloatingOrPathLikeCalibrationID(t *testing.T) {
	for _, bad := range []string{"latest", "../../x", "cal-1"} {
		file := config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": {CalibrationID: bad}}}
		require.Error(t, file.Validate(), bad)
	}
	file := config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": {CalibrationID: "cal-20261004-0a1b2c3d", CalibrationMaxAgeDays: 30}}}
	require.NoError(t, file.Validate())
	assert.Equal(t, 30*24*time.Hour, file.Providers["jev"].CalibrationMaxAge())
	file.Providers["jev"] = config.Provider{CalibrationMaxAgeDays: -1}
	require.Error(t, file.Validate())
}

func TestPinRefusesARecordOfAnotherProfileThanTheProvidersAndFollowsTheProvider(t *testing.T) {
	root := calibrationRig(t)
	v1 := publish(t, root, 0.84, nil)
	v2 := publish(t, root, 0.84, func(m *calibration.Meta, _ *calibration.Report) {
		m.Profile = calibration.ProfileV2
		m.ProfileVersion = handoffconsumer.ProfileVersionOf(calibration.ProfileV2, calibration.BoundaryPackage)
		m.Seed = 7
	})

	_, err := runPin(t, root, v2.ID, true)
	require.ErrorContains(t, err, "another question profile", "the provider is on v1")
	assert.Empty(t, pinnedID(t, root))

	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	provider := file.Providers["jev"]
	provider.Profile = calibration.ProfileV2
	file.Providers["jev"] = provider
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), file))

	_, err = runPin(t, root, v1.ID, true)
	require.Error(t, err, "now the v1 record is the wrong one")
	_, err = runPin(t, root, v2.ID, true)
	require.NoError(t, err)
	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "calibration: calibrado ("+v2.ID)
}

func TestAnAtomicRecordPinsOnAnAtomicProviderAndTheDoctorSaysCalibrated(t *testing.T) {
	root := calibrationRig(t)
	atomic := publish(t, root, 0.62, func(m *calibration.Meta, _ *calibration.Report) {
		m.Profile = calibration.ProfileAtomicV1
		m.ProfileVersion = handoffconsumer.ProfileVersionOf(calibration.ProfileAtomicV1, calibration.BoundaryPackage)
		m.Seed = 9
	})
	_, err := runPin(t, root, atomic.ID, true)
	require.ErrorContains(t, err, "another question profile", "the provider is still on v1")

	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	provider := file.Providers["jev"]
	provider.Profile = calibration.ProfileAtomicV1
	file.Providers["jev"] = provider
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), file))

	_, err = runPin(t, root, atomic.ID, true)
	require.NoError(t, err)
	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	assert.Contains(t, out, "calibration: calibrado ("+atomic.ID)
}

func TestShowPrintsTheCrossValidationAndTheCriterionReadings(t *testing.T) {
	root := calibrationRig(t)
	record := publish(t, root, 0.62, func(m *calibration.Meta, r *calibration.Report) {
		m.Profile, m.ProfileVersion, m.Seed = calibration.ProfileAtomicV1, handoffconsumer.ProfileVersionOf(calibration.ProfileAtomicV1, calibration.BoundaryPackage), 11
		r.HeldOut = nil
		r.CrossValidation = &calibration.CrossValidation{Folds: 5, FalseApproval: calibration.NewRate(0, 150), FalseRejection: calibration.NewRate(0, 30)}
		r.CriterionCalibration = []calibration.CriterionResult{
			{Criterion: "acceptance_checkable", Status: calibration.CriterionReliable, AUC: 1, Threshold: 0.46, FalseFlag: calibration.NewRate(1, 21), Detection: calibration.NewRate(21, 21)},
			{Criterion: "validation_stated", Status: calibration.CriterionUnreliable, AUC: 0.84, Threshold: 0.05, Note: "auc 0.840 below 0.90"},
		}
	})

	out, err := runCalibrationCommand(t, func(c *cobraCmd) error { return runCalibrationShow(c, integrationsOptions{Root: root}, record.ID) })

	require.NoError(t, err)
	assert.Contains(t, out, "recommended threshold: 0.62 (cross-validation over 5 folds")
	assert.Contains(t, out, "criterion acceptance_checkable: reliable (auc 1.000")
	assert.Contains(t, out, "criterion validation_stated: unreliable")
	assert.Contains(t, out, "auc 0.840 below 0.90")
}
