// Package jev adapts the documented JEV typed-question API (POST /v1/systemone)
// to the integration contract. Everything here follows the shapes in the
// official quickstart; behavior the documentation does not show (error bodies,
// limits, idempotency) is never assumed.
package jev

import (
	"encoding/json"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

type wireQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type wireRequest struct {
	State     string                  `json:"state"`
	Model     string                  `json:"model"`
	Questions map[string]wireQuestion `json:"questions"`
}

func invalidRequest(reason string) error {
	return integration.NewError(integration.StateIncompatibleContract, integration.Correlation{Provider: providerName}, reason)
}

func encodeRequest(model, state string, questions []integration.Question) ([]byte, error) {
	if len(questions) == 0 {
		return nil, invalidRequest("at least one question is required")
	}
	wire := wireRequest{State: state, Model: model, Questions: map[string]wireQuestion{}}
	for _, question := range questions {
		encoded, err := encodeQuestion(question)
		if err != nil {
			return nil, err
		}
		if _, dup := wire.Questions[question.Key]; dup {
			return nil, invalidRequest("question keys must be unique")
		}
		wire.Questions[question.Key] = encoded
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, invalidRequest("request could not be encoded")
	}
	return body, nil
}

func encodeQuestion(q integration.Question) (wireQuestion, error) {
	if q.Key == "" || q.Instructions == "" {
		return wireQuestion{}, invalidRequest("a question needs a key and instructions")
	}
	out := wireQuestion{Type: string(q.Kind), Instructions: q.Instructions}
	switch q.Kind {
	case integration.KindNoul:
		return out, nil
	case integration.KindChoice:
		return withCriteria(out, optionCriteria(q.Options), len(q.Options))
	case integration.KindScore:
		return withCriteria(out, q.Labels, len(q.Labels))
	default:
		return wireQuestion{}, invalidRequest(fmt.Sprintf("question kind %q is not supported", q.Kind))
	}
}

func withCriteria(q wireQuestion, criteria any, count int) (wireQuestion, error) {
	if count == 0 {
		return wireQuestion{}, invalidRequest("a choice or score question needs criteria")
	}
	q.Criteria = criteria
	return q, nil
}

func optionCriteria(options []integration.Option) map[string]string {
	criteria := make(map[string]string, len(options))
	for _, option := range options {
		criteria[option.Key] = option.Description
	}
	return criteria
}
