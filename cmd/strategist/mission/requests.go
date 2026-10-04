package mission

import (
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

// NewRequests builds `mission requests`, a read-only listing of embedded
// invocation requests. It shows state and expiry so an agent can tell which
// request is still usable; it never prints a payload, input or nonce.
func NewRequests(deps InvocationDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests",
		Short: "List embedded invocation requests (read-only; never prints payload or nonce)",
	}
	var rootInput, missionID string
	var asJSON bool
	cmd.Flags().StringVar(&rootInput, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().StringVar(&missionID, "mission-id", "", "only list the requests of this mission (default: every mission)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunRequests(cmd, deps, rootInput, missionID, asJSON)
	}
	return cmd
}

// RunRequests lists the request records below the resolved Strategist root.
func RunRequests(cmd *cobra.Command, deps InvocationDependencies, rootInput, missionID string, asJSON bool) error {
	if missionID != "" {
		if err := deps.RequireMissionID(missionID); err != nil {
			return err
		}
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission requests: %w", err)
	}
	listing, err := deps.ListRequests(root, missionID)
	if err != nil {
		return fmt.Errorf("mission requests: %w", err)
	}
	if asJSON {
		return deps.WriteResult(cmd, true, listing)
	}
	return writeRequestsTable(cmd, listing)
}

func writeRequestsTable(cmd *cobra.Command, listing domain.MissionInvocationListing) error {
	var out strings.Builder
	if len(listing.Requests) == 0 {
		out.WriteString("no invocation requests\n")
	} else {
		out.WriteString("request_id\tmission_id\trole\tslot\tstate\tcreated_at\texpires_at\texpired\n")
	}
	for _, request := range listing.Requests {
		fmt.Fprintf(&out, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%t\n", request.RequestID, request.MissionID, request.Role, request.Slot,
			request.State, request.CreatedAt.UTC().Format(time.RFC3339), request.ExpiresAt.UTC().Format(time.RFC3339), request.Expired)
	}
	if len(listing.Skipped) > 0 {
		fmt.Fprintf(&out, "unreadable records skipped: %s\n", strings.Join(listing.Skipped, ", "))
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), out.String()); err != nil {
		return fmt.Errorf("write mission requests: %w", err)
	}
	return nil
}
