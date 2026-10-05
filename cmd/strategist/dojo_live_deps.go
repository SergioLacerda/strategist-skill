package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/live"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
)

// calibrationEndpoint is the endpoint a run would call, shown in the consent term.
func calibrationEndpoint(root string) string {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	if provider, ok := file.Providers[integrationProvider]; err == nil && found && ok && provider.Endpoint != "" {
		return provider.Endpoint
	}
	return setup.DefaultEndpoint
}

func livePlan() (calls, bytes int) { return live.Calls, live.Bytes() }

// runLiveSmoke runs the live smoke check against the real provider. It is only
// reachable from a DOJO run that carries the operator's consent.
func runLiveSmoke(ctx context.Context, root string) ([]domain.DojoCheckItem, error) {
	file, found, err := config.Load(filepath.Join(root, config.FileName))
	provider, ok := file.Providers[integrationProvider]
	if err != nil || !found || !ok || !provider.Enabled {
		return nil, errors.New("no provider integration is enabled: run `strategist integrations enable` and `strategist integrations doctor` first")
	}
	adapter, env, err := newJEVAdapter(root, provider)
	if err != nil {
		return nil, fmt.Errorf("live smoke: %w", err)
	}
	judge, err := calibrationJudge(root, provider.Profile)
	if err != nil {
		return nil, fmt.Errorf("live smoke: %w", err)
	}
	checks := live.Smoke(ctx, live.Env{
		Probe: func(c context.Context) probe.Record { return probe.Run(c, provider, adapter, time.Now) },
		Judge: judge,
		Secret: func() string {
			secret, resolveErr := credential.Resolve(provider.CredentialRef, env)
			if resolveErr != nil {
				return ""
			}
			return secret.Reveal()
		},
	})
	items := make([]domain.DojoCheckItem, len(checks))
	for i, c := range checks {
		items[i] = domain.DojoCheckItem{Label: c.Name, Passed: c.Passed, Detail: c.Detail}
	}
	return items, nil
}
