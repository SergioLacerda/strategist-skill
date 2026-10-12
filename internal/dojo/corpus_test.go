package dojo_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCorpusIncludesCompletePackagesAndAcceptedRefinedOnly(t *testing.T) {
	w := newWorkspace(t)
	w.writePackage(t, "done", "finished", 1, "documentation_applied")
	w.writePackage(t, "refined", "accepted", 2, "gate_analysis_accepted")
	w.writePackage(t, "refined", "waiting", 3, "archivist_done")
	incomplete := w.writePackage(t, "done", "incomplete", 4, "documentation_applied")
	require.NoError(t, os.Remove(filepath.Join(incomplete, "proposal.md")))

	corpus, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})

	require.NoError(t, err)
	names := []string{}
	for _, e := range corpus.Manifest.Entries {
		names = append(names, e.Name)
		assert.Equal(t, "candidate", e.Label, "an unreviewed package is only a candidate")
		assert.True(t, e.WeakLabel)
		assert.NotContains(t, e.Path, w.base, "the manifest stores references, not absolute paths")
	}
	assert.ElementsMatch(t, []string{"finished", "accepted"}, names)
	assert.Len(t, corpus.Sources, 2)
}

func TestBuildCorpusIsDeterministicAndDigestsFollowContent(t *testing.T) {
	w := newWorkspace(t)
	w.packages(t, 5)
	first, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	second, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.Equal(t, first.Manifest, second.Manifest)

	w.writePackage(t, "done", "pkg-000", 99, "documentation_applied")
	changed, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.NotEqual(t, first.Manifest.Digest, changed.Manifest.Digest)
}

func TestBuildCorpusSplitsHeldOutByDigest(t *testing.T) {
	w := newWorkspace(t)
	w.packages(t, 60)
	cal := policyCase()
	cal.Corpus.HeldOutPercent = 25
	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	held := 0
	for _, e := range corpus.Manifest.Entries {
		assert.Equal(t, calibration.HeldOut(e.Digest, "", 25), e.HeldOut)
		if e.HeldOut {
			held++
		}
	}
	assert.InDelta(t, 15, held, 9)
}

func TestEveryCaseInheritsTheHeldOutSideOfItsPackage(t *testing.T) {
	w := newWorkspace(t)
	w.packages(t, 30)
	cal := policyCase()
	cal.Corpus.HeldOutPercent = 25
	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	side := map[string]bool{}
	for _, src := range corpus.Sources {
		side[src.Name] = src.HeldOut
	}
	for _, c := range dojo.BuildCases(corpus, cal) {
		assert.Equal(t, side[c.Source], c.HeldOut, c.ID)
	}
}

func TestBuildCorpusHonorsIncludeAndCustomRoots(t *testing.T) {
	w := newWorkspace(t)
	w.writePackage(t, "done", "keep", 1, "documentation_applied")
	w.writePackage(t, "done", "skip", 2, "documentation_applied")
	w.writePackage(t, "extra", "other", 3, "documentation_applied")
	cal := policyCase()
	cal.Corpus.Include = []string{"keep"}
	corpus, err := dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	require.Len(t, corpus.Manifest.Entries, 1)
	assert.Equal(t, "keep", corpus.Manifest.Entries[0].Name)

	cal = policyCase()
	cal.Corpus.Roots = []string{"extra"}
	corpus, err = dojo.BuildCorpus(w.base, cal, handoffconsumer.Reader{})
	require.NoError(t, err)
	require.Len(t, corpus.Manifest.Entries, 1)
	assert.Equal(t, "extra", corpus.Manifest.Entries[0].Status)
}

func TestPromotionMakesAPackageGoldenOnlyForItsDigest(t *testing.T) {
	w := newWorkspace(t)
	dir := w.writePackage(t, "done", "good", 1, "documentation_applied")
	corpus, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	digest := corpus.Manifest.Entries[0].Digest

	require.Error(t, dojo.AppendPromotion(w.base, dojo.Promotion{Digest: digest, Package: "good"}), "a reviewer and an oracle are required")
	require.NoError(t, dojo.AppendPromotion(w.base, dojo.Promotion{Digest: digest, Package: "good", ReviewedBy: "me", Oracle: "reads clean"}))
	require.Error(t, dojo.AppendPromotion(w.base, dojo.Promotion{Digest: digest, Package: "good", ReviewedBy: "me", Oracle: "again"}))

	corpus, err = dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.Equal(t, "golden", corpus.Manifest.Entries[0].Label)
	assert.False(t, corpus.Manifest.Entries[0].WeakLabel)
	assert.True(t, corpus.Sources[0].Golden)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(fmtPlan(7)), 0o644))
	corpus, err = dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.Equal(t, "candidate", corpus.Manifest.Entries[0].Label, "editing the package voids the promotion")
}

func fmtPlan(variant int) string { return fmt.Sprintf(planTemplate, variant) }

func TestAmendmentBeforeSnapshotsBecomeNaturalNegatives(t *testing.T) {
	w := newWorkspace(t)
	dir := w.writePackage(t, "done", "revised", 1, "documentation_applied")
	snapshot := filepath.Join(dir, ".amendments", "001")
	require.NoError(t, os.MkdirAll(snapshot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(snapshot, "tasks.md"), []byte("- [ ] 1.1 Do something vague\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".amendments", "002"), 0o755)) // no tasks: skipped
	corpus, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)

	require.Len(t, corpus.Befores, 1)
	assert.Equal(t, 2, corpus.Manifest.Entries[0].Amendments)
	var natural []calibration.Case
	for _, c := range dojo.BuildCases(corpus, policyCase()) {
		if c.Kind == calibration.KindNatural {
			natural = append(natural, c)
		}
	}
	require.Len(t, natural, 1)
	assert.Equal(t, calibration.ExpectReject, natural[0].Expect)
	assert.NotEmpty(t, natural[0].ParentID)
}

func TestCandidatesFlagButNeverPromote(t *testing.T) {
	w := newWorkspace(t)
	w.writePackage(t, "done", "flagged", 1, "documentation_applied")
	require.NoError(t, dojo.AppendCandidate(w.base, dojo.Candidate{Mission: "flagged", Package: "flagged", Transition: "archivist_to_gate", Kind: "provider_rejected_accepted_package", Confidence: 0.4, Threshold: 0.8, Outcome: "accepted"}))
	require.Error(t, dojo.AppendCandidate(w.base, dojo.Candidate{}))
	corpus, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.Equal(t, "candidate", corpus.Manifest.Entries[0].Label)
	assert.Equal(t, "provider_rejected_accepted_package", corpus.Manifest.Entries[0].ReviewFlag)
}

func TestMalformedLabelLinesAreReportedNotDropped(t *testing.T) {
	w := newWorkspace(t)
	dir := filepath.Join(w.base, "dojo", dojo.DirCorpus)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "golden.jsonl"), []byte("not json\n{\"digest\":\"d\"}\n"), 0o644))
	labels, err := dojo.LoadLabels(w.base)
	require.NoError(t, err)
	assert.Equal(t, 2, labels.Malformed)
}

func TestADemotionWithdrawsAPromotionAndALaterPromotionRestoresIt(t *testing.T) {
	w := newWorkspace(t)
	w.writePackage(t, "done", "good", 1, "documentation_applied")
	corpus, err := dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	digest := corpus.Manifest.Entries[0].Digest

	require.Error(t, dojo.AppendDemotion(w.base, dojo.Demotion{Digest: digest, Package: "good", ReviewedBy: "me", Reason: "x"}), "only a golden digest can be demoted")
	require.NoError(t, dojo.AppendPromotion(w.base, dojo.Promotion{Digest: digest, Package: "good", ReviewedBy: "me", Oracle: "reads clean", At: "2026-10-05T10:00:00Z"}))

	require.Error(t, dojo.AppendDemotion(w.base, dojo.Demotion{Digest: digest, Package: "good", ReviewedBy: "me"}), "a reason is required")
	require.NoError(t, dojo.AppendDemotion(w.base, dojo.Demotion{Digest: digest, Package: "good", ReviewedBy: "me", Reason: "tasks state no verification", At: "2026-10-05T11:00:00Z"}))
	corpus, err = dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.Equal(t, "candidate", corpus.Manifest.Entries[0].Label, "a demoted package is a candidate again")
	require.Error(t, dojo.AppendDemotion(w.base, dojo.Demotion{Digest: digest, Package: "good", ReviewedBy: "me", Reason: "again"}), "it is no longer golden")

	require.NoError(t, dojo.AppendPromotion(w.base, dojo.Promotion{Digest: digest, Package: "good", ReviewedBy: "me", Oracle: "reviewed again", At: "2026-10-05T12:00:00Z"}))
	corpus, err = dojo.BuildCorpus(w.base, policyCase(), handoffconsumer.Reader{})
	require.NoError(t, err)
	assert.Equal(t, "golden", corpus.Manifest.Entries[0].Label, "a promotion after the demotion wins")
}
