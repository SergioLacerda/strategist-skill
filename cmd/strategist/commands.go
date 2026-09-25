package main

import (
	dojoadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/dojo"
	evaladapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/eval"
	installadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/install"
	levelingadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/leveling"
	metricsadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/metrics"
	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	pluginsadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/plugins"
	"github.com/SergioLacerda/strategist-skill/internal/check"
	"github.com/SergioLacerda/strategist-skill/treasure-chest/cli"
	"github.com/spf13/cobra"
)

// registerCommands is the single place where every top-level command is
// attached to root. Command families own their flags and behavior; this
// function only composes them. Cobra sorts sub-commands by name, so the order of
// the calls below does not affect help output.
func registerCommands(root *cobra.Command) {
	installadapter.Register(root, installDependencies(), upgradeDependencies())
	root.AddCommand(compileCmd, validateCmd, syncGovernanceCmd, versionCmd, providerCmd, handoffCmd, mechanismsCmd)
	check.Register(root)
	treasurecli.Register(root)
	metricsadapter.Register(root, metricsDependencies(), roleLevelLedger, defaultLedgerMaxRecords)
	missionadapter.Register(root, missionLifecycleDependencies(), missionViewDependencies(), missionNormalizeDependencies(), missionReportUsageDependencies())
	pluginsadapter.Register(root)
	dojoadapter.Register(root, dojoDependencies())
	levelingadapter.Register(root, levelingAdapterDependencies())
	evaladapter.Register(root, evalDependencies(), evalHarvestDependencies())
}
