package mission

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func requestsListing() domain.MissionInvocationListing {
	created := time.Date(2026, 10, 4, 12, 53, 57, 0, time.UTC)
	return domain.MissionInvocationListing{Requests: []domain.MissionInvocationSummary{
		{RequestID: "inv_expired01", MissionID: "m-a", Role: "ranger", Slot: "discovery", State: domain.InvocationStatePending, CreatedAt: created, ExpiresAt: created.Add(time.Hour), Expired: true},
		{RequestID: "inv_done0001", MissionID: "m-a", Role: "ranger", Slot: "discovery", State: domain.InvocationStateCompleted, CreatedAt: created.Add(8 * time.Second), ExpiresAt: created.Add(time.Hour)},
	}}
}

func requestsDependencies(listing domain.MissionInvocationListing, gotRoot, gotMission *string) InvocationDependencies {
	deps := invocationTestDependencies(domain.MissionInvocationRequest{})
	deps.ListRequests = func(root, missionID string) (domain.MissionInvocationListing, error) {
		if gotRoot != nil {
			*gotRoot = root
		}
		if gotMission != nil {
			*gotMission = missionID
		}
		return listing, nil
	}
	return deps
}

func TestRequestsEmitsTheListingAsJSONAndFiltersByMission(t *testing.T) {
	var root, mission string
	cmd := NewRequests(requestsDependencies(requestsListing(), &root, &mission))
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--root", "strategist-root", "--mission-id", "m-a", "--json"})

	require.NoError(t, cmd.Execute())

	require.Equal(t, "strategist-root", root)
	require.Equal(t, "m-a", mission)
	var got domain.MissionInvocationListing
	require.NoError(t, json.Unmarshal(output.Bytes(), &got))
	require.Equal(t, []string{"inv_expired01", "inv_done0001"}, []string{got.Requests[0].RequestID, got.Requests[1].RequestID})
	require.True(t, got.Requests[0].Expired)
}

func TestRequestsHumanOutputIsOneLinePerRequestWithoutJSON(t *testing.T) {
	cmd := NewRequests(requestsDependencies(requestsListing(), nil, nil))
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--mission-id", "m-a"})

	require.NoError(t, cmd.Execute())

	text := output.String()
	require.Contains(t, text, "request_id\tmission_id\trole\tslot\tstate\tcreated_at\texpires_at\texpired")
	require.Contains(t, text, "inv_expired01\tm-a\tranger\tdiscovery\tpending\t2026-10-04T12:53:57Z\t2026-10-04T13:53:57Z\ttrue")
	require.Contains(t, text, "inv_done0001\tm-a\tranger\tdiscovery\tcompleted\t2026-10-04T12:54:05Z\t2026-10-04T13:53:57Z\tfalse")
}

func TestRequestsMissionIsOptionalAndNotValidatedWhenOmitted(t *testing.T) {
	var mission string
	deps := requestsDependencies(requestsListing(), nil, &mission)
	deps.RequireMissionID = func(string) error { return errors.New("mission id must not be required for a listing") }
	cmd := NewRequests(deps)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	require.NoError(t, cmd.Execute())
	require.Empty(t, mission, "no filter means every mission")
}

func TestRequestsSaysSoWhenThereAreNone(t *testing.T) {
	cmd := NewRequests(requestsDependencies(domain.MissionInvocationListing{}, nil, nil))
	var output bytes.Buffer
	cmd.SetOut(&output)

	require.NoError(t, cmd.Execute())
	require.Contains(t, output.String(), "no invocation requests")
}

func TestRequestsReportsSkippedRecordsByNameInHumanOutput(t *testing.T) {
	listing := requestsListing()
	listing.Skipped = []string{"inv_corrupt01.json"}
	cmd := NewRequests(requestsDependencies(listing, nil, nil))
	var output bytes.Buffer
	cmd.SetOut(&output)

	require.NoError(t, cmd.Execute())
	require.Contains(t, output.String(), "unreadable records skipped: inv_corrupt01.json")
}

func TestRequestsWrapsListingErrors(t *testing.T) {
	deps := invocationTestDependencies(domain.MissionInvocationRequest{})
	deps.ListRequests = func(string, string) (domain.MissionInvocationListing, error) {
		return domain.MissionInvocationListing{}, errors.New("disk failure")
	}
	cmd := NewRequests(deps)
	cmd.SetOut(&bytes.Buffer{})

	require.ErrorContains(t, cmd.Execute(), "mission requests: disk failure")
}
