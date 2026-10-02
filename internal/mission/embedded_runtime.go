package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// NewEmbeddedWeaponDispatch builds the production Ranked dispatch from the
// compiled registry and an explicitly supplied host-agent prompt bridge. The
// payload source is always the CLI's embedded defaults filesystem.
func NewEmbeddedWeaponDispatch(registry domain.CompiledRegistry, bridge connectors.EmbeddedPromptBridge, codeInvokers map[string]connectors.EmbeddedWeaponInvoker) (connectors.EmbeddedWeaponDispatch, error) {
	dispatch, err := connectors.NewProductionEmbeddedDispatch(registry, connectors.EmbeddedRuntimeDependencies{
		CodeInvokers: codeInvokers,
		PromptBridge: bridge,
		Payloads: func(_ context.Context, weaponID, version string) (connectors.EmbeddedPromptPayload, error) {
			payload, digest, err := (strategistembed.Extractor{}).ReadEmbeddedWeaponPayload(weaponID, version)
			if err != nil {
				return connectors.EmbeddedPromptPayload{}, fmt.Errorf("read embedded Weapon payload %q: %w", domain.WeaponIdentity(weaponID, version), err)
			}
			return connectors.EmbeddedPromptPayload{WeaponID: weaponID, Content: payload, SourceDigest: digest}, nil
		},
	})
	if err != nil {
		return connectors.EmbeddedWeaponDispatch{}, fmt.Errorf("build embedded Weapon dispatch: %w", err)
	}
	return dispatch, nil
}
