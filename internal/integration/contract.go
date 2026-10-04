package integration

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// ContractVersion identifies the integration contract adapters implement.
const ContractVersion = "strategist.integration/v1"

// Capability is a versioned operation a consumer asks of a provider.
type Capability string

// Capabilities known to the host contract. A provider may support a subset.
const (
	CapHandoffValidate    Capability = "handoff.validate"
	CapHandoffProject     Capability = "handoff.project"
	CapConfidenceEvaluate Capability = "confidence.evaluate"
)

// QuestionKind is the typed question vocabulary of the documented provider.
type QuestionKind string

// Question kinds.
const (
	KindNoul   QuestionKind = "noul"
	KindChoice QuestionKind = "choice"
	KindScore  QuestionKind = "score"
)

// Option is one labelled criterion of a choice question.
type Option struct {
	Key         string
	Description string
}

// Question is one typed criterion evaluated against a state text. Options
// apply to a choice, Labels (ordered) to a score.
type Question struct {
	Key          string
	Kind         QuestionKind
	Instructions string
	Options      []Option
	Labels       []string
}

// Request is a capability call. State is free text sent to a third party, so
// the consumer projects it through its data policy before it gets here.
type Request struct {
	Correlation
	Capability Capability
	State      string
	Questions  []Question
}

// Answer is one typed provider answer. Confidence is meaningful only when
// HasConfidence is true: a noul answer never carries one.
type Answer struct {
	Kind          QuestionKind
	Noul          float64
	Choice        string
	Score         float64
	Confidence    float64
	HasConfidence bool
}

// Usage is the provider-reported token usage of one call.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Result is the typed outcome of a call. Model is the resolved model, which is
// part of the binding identity.
type Result struct {
	Answers map[string]Answer
	Model   string
	Usage   Usage
}

// Identity is what an approved adapter declares about itself.
type Identity struct {
	Provider     string
	Version      string
	Contract     string
	Capabilities []Capability
}

// Adapter is the port a provider implements. It translates a capability call
// into the provider's protocol; it never decides a binding or a path.
type Adapter interface {
	Identity() Identity
	Call(ctx context.Context, request Request) (Result, error)
}

// Binding is the explicit selection of one adapter version for a consumer and
// capability. Model is the resolved provider model, not an alias.
type Binding struct {
	Provider   string
	Version    string
	Model      string
	Consumer   string
	Capability Capability
}

// Digest is the deterministic identity of the binding.
func (b Binding) Digest() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s", ContractVersion, b.Provider, b.Version, b.Model, b.Consumer, b.Capability)))
	return fmt.Sprintf("sha256:%x", sum)
}
