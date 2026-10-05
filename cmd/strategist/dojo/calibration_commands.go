package dojo

import "github.com/spf13/cobra"

func calibrationCommands(deps Dependencies) []*cobra.Command {
	return []*cobra.Command{
		newCalibrate(deps), newGenerate(deps), newPromote(deps), newCandidate(deps), newAudit(deps), newPropose(deps), newCompare(deps),
	}
}
