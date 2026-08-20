// Package sink selects and wraps concrete telemetry.EventSink
// implementations. It lives one level above its noop/slog/jsonl/otel/external
// subpackages (which only depend on internal/telemetry) so that Select can
// import all of them without creating an import cycle back into
// internal/telemetry itself.
package sink

import (
	"context"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/governancebridge"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	externalsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/external"
	otelsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/otel"
	slogsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/slog"
)

// Select implements the sink-selection policy (item 1 §7 of the
// governança-plugável document): no OTel endpoint configured -> local slog
// sink; OTel endpoint configured -> otel sink; a GovernanceBridge is present
// -> the chosen sink is additionally wrapped so events also reach external
// governance (acceptance check 6.1: bridge == nil must never break this —
// the external wrap is only added when bridge is non-nil). The result is
// always wrapped in the strict/non-strict resilience policy from cfg.Strict
// (item 5).
func Select(cfg telemetry.Config, bridge governancebridge.GovernanceBridge) telemetry.EventSink {
	var base telemetry.EventSink
	if cfg.Enabled() {
		base = otelsink.New()
	} else {
		base = slogsink.New()
	}
	if bridge != nil {
		base = externalsink.New(base)
	}
	return Resilient(base, cfg.Strict)
}

// Resilient wraps inner with Strategist's exporter-failure policy (item 5):
// strict=false (default) logs and swallows an Emit error — telemetry failure
// never blocks a mission (fail-open, matching the OTel SDK's own
// error-handling spec). strict=true propagates the error unchanged
// (fail-closed), for staging/CI environments that want to catch
// misconfiguration early.
func Resilient(inner telemetry.EventSink, strict bool) telemetry.EventSink {
	return resilientSink{inner: inner, strict: strict}
}

type resilientSink struct {
	inner  telemetry.EventSink
	strict bool
}

func (r resilientSink) Emit(ctx context.Context, event telemetry.Event) error {
	err := r.inner.Emit(ctx, event)
	if err == nil {
		return nil
	}
	if r.strict {
		return err
	}
	slog.WarnContext(ctx, "telemetry: event emit failed (non-blocking)", "error", err, "event.name", event.Name)
	return nil
}

var _ telemetry.EventSink = resilientSink{}
