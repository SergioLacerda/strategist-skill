package jev

import (
	"bytes"
	"context"
	"errors"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/transport"
)

const (
	providerName   = "jev"
	floatingAlias  = "jev-latest"
	adapterVersion = "1"
)

// Config is the operator-pinned adapter configuration.
type Config struct {
	// Model is the pinned provider model, for example "jev-1.13.0".
	Model string
	// AllowAlias lets Model be the floating "jev-latest" alias. Without it the
	// alias is rejected, because a floating model changes behavior silently.
	AllowAlias bool
	// Version overrides the adapter version used for binding resolution.
	Version string
}

// Adapter implements integration.Adapter for the documented JEV API.
type Adapter struct {
	client *transport.Client
	secret func() (credential.Secret, error)
	cfg    Config
}

// New builds the adapter. The secret is resolved per call so a rotated key is
// picked up without rebuilding or rebinding anything.
func New(client *transport.Client, secret func() (credential.Secret, error), cfg Config) (*Adapter, error) {
	if cfg.Model == "" {
		return nil, errors.New("jev adapter requires a pinned model")
	}
	if cfg.Model == floatingAlias && !cfg.AllowAlias {
		return nil, errors.New("jev-latest is a floating alias; pin a model or opt in explicitly")
	}
	if cfg.Version == "" {
		cfg.Version = adapterVersion
	}
	return &Adapter{client: client, secret: secret, cfg: cfg}, nil
}

// Identity declares what the adapter is and supports. handoff.project is not
// listed: the documented API cannot generate a structured body.
func (a *Adapter) Identity() integration.Identity {
	return integration.Identity{
		Provider:     providerName,
		Version:      a.cfg.Version,
		Contract:     integration.ContractVersion,
		Capabilities: []integration.Capability{integration.CapHandoffValidate, integration.CapConfidenceEvaluate},
	}
}

// Call translates the capability into typed questions, posts them and returns
// the validated typed result with the resolved model.
func (a *Adapter) Call(ctx context.Context, request integration.Request) (integration.Result, error) {
	correlation := request.Correlation
	correlation.Provider = providerName
	result, err := a.call(ctx, correlation, request)
	if err != nil {
		return integration.Result{}, integration.WithCorrelation(err, correlation) //nolint:wrapcheck // typed integration error; wrapping would only repeat its text
	}
	return result, nil
}

func (a *Adapter) call(ctx context.Context, correlation integration.Correlation, request integration.Request) (integration.Result, error) {
	body, err := a.prepare(correlation, request)
	if err != nil {
		return integration.Result{}, err
	}
	result, err := a.send(ctx, correlation, body, request.Questions)
	if err != nil {
		return integration.Result{}, err
	}
	if a.cfg.Model != floatingAlias && result.Model != a.cfg.Model {
		return integration.Result{}, integration.NewError(integration.StateBindingIntegrity, correlation, "resolved model differs from the pinned model")
	}
	return result, nil
}

// prepare checks the capability and encodes the request before any secret is read.
func (a *Adapter) prepare(correlation integration.Correlation, request integration.Request) ([]byte, error) {
	if request.Capability != integration.CapHandoffValidate && request.Capability != integration.CapConfidenceEvaluate {
		return nil, integration.NewError(integration.StateUnsupportedCapability, correlation, "capability is not supported by the documented API")
	}
	return encodeRequest(a.cfg.Model, request.State, request.Questions)
}

func (a *Adapter) send(ctx context.Context, correlation integration.Correlation, body []byte, asked []integration.Question) (integration.Result, error) {
	secret, err := a.secret()
	if err != nil {
		return integration.Result{}, err
	}
	response, err := a.client.Post(ctx, correlation, secret, body)
	if err != nil {
		return integration.Result{}, err //nolint:wrapcheck // typed integration error from the transport
	}
	if bytes.Contains(response.Body, []byte(secret.Reveal())) {
		return integration.Result{}, invalid("response echoes the credential")
	}
	return decodeResponse(response.Body, asked)
}
