package policy

import (
	"slices"
	"sync"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// Breaker opens after a run of provider failures so a down provider is not hit
// on every attempt. It counts only failures that say something about provider
// health; local ones (integrity, data policy, missing credential) are ignored.
type Breaker struct {
	mu        sync.Mutex
	threshold int
	cooldown  time.Duration
	now       func() time.Time
	failures  int
	openedAt  time.Time
}

// NewBreaker builds a closed breaker.
func NewBreaker(threshold int, cooldown time.Duration, now func() time.Time) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown, now: now}
}

var countableStates = []integration.State{
	integration.StateAuthenticationFailed, integration.StateForbidden, integration.StateTimeout,
	integration.StateUnavailable, integration.StateRateLimited, integration.StateInvalidResponse,
}

func countable(state integration.State) bool { return slices.Contains(countableStates, state) }

// Check returns unavailable while the circuit is open and no state otherwise.
// After the cooldown it lets calls through again (half-open); the next Record
// decides whether it closes or reopens.
func (b *Breaker) Check() integration.State {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures >= b.threshold && b.now().Before(b.openedAt.Add(b.cooldown)) {
		return integration.StateUnavailable
	}
	return ""
}

// Record feeds the outcome of a call: nil closes the circuit, a countable
// failure counts toward opening it, anything else is ignored.
func (b *Breaker) Record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.failures = 0
		return
	}
	state, ok := integration.StateOf(err)
	if !ok || !countable(state) {
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.openedAt = b.now()
	}
}
