package dojo

import (
	"fmt"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/spf13/cobra"
)

func newCompare(deps Dependencies) *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "compare <case-a> <case-b>",
		Short: "Compare the latest observations of two cases, offline (a laboratory reading)",
		Long: `Print, per operator and level, the mean probability of accept of two cases side by side, the AUC
and an exploratory operating sweep. It reads stored observations only: no call, nothing persisted, no
calibration and no recommendation. Use it to see what a change of profile or operator set did.`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := latestObservations(deps, root, args[0])
			if err != nil {
				return err
			}
			b, err := latestObservations(deps, root, args[1])
			if err != nil {
				return err
			}
			return internaldojo.Compare(args[0], args[1], a, b).Render(cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&root, "root", "", rootFlagUsage)
	return cmd
}

func latestObservations(deps Dependencies, root, name string) ([]calibration.Observation, error) {
	cc, err := loadCase(deps, root, name)
	if err != nil {
		return nil, err
	}
	observed, _, err := internaldojo.ReadObservations(cc.paths.ObservationsPath)
	if err != nil {
		return nil, fmt.Errorf("dojo compare: %w", err)
	}
	if len(observed) == 0 {
		return nil, fmt.Errorf("dojo compare: case %q has no observations yet", name)
	}
	return calibration.SessionOf(observed, calibration.LatestSession(observed)), nil
}
