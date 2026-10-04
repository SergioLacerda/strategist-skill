package jev

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

type wireAnswer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul"`
	Choice        *string            `json:"choice"`
	Score         *float64           `json:"score"`
	Confidence    *float64           `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
	Legend        map[string]string  `json:"legend"`
}

type wireResponse struct {
	Model   string                `json:"model"`
	Answers map[string]wireAnswer `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func invalid(reason string) error {
	return integration.NewError(integration.StateInvalidResponse, integration.Correlation{Provider: providerName}, reason)
}

// decodeResponse parses a provider body strictly against the questions asked:
// unknown fields, missing or extra answers and shapes that do not match the
// question type are rejected. Free text from the provider is never carried
// into the result except values checked against the asked options.
func decodeResponse(body []byte, asked []integration.Question) (integration.Result, error) {
	var wire wireResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return integration.Result{}, invalid("response is not the documented JSON shape")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return integration.Result{}, invalid("response has trailing content")
	}
	if !modelName.MatchString(wire.Model) || wire.Usage.InputTokens < 0 || wire.Usage.OutputTokens < 0 {
		return integration.Result{}, invalid("response model or usage is invalid")
	}
	answers, err := convertAnswers(wire.Answers, asked)
	if err != nil {
		return integration.Result{}, err
	}
	return integration.Result{
		Answers: answers,
		Model:   wire.Model,
		Usage:   integration.Usage{InputTokens: wire.Usage.InputTokens, OutputTokens: wire.Usage.OutputTokens},
	}, nil
}

func convertAnswers(wire map[string]wireAnswer, asked []integration.Question) (map[string]integration.Answer, error) {
	if len(wire) != len(asked) {
		return nil, invalid("answers do not match the questions asked")
	}
	answers := make(map[string]integration.Answer, len(asked))
	for _, question := range asked {
		raw, ok := wire[question.Key]
		if !ok {
			return nil, invalid("an asked question has no answer")
		}
		answer, err := convertAnswer(question, raw)
		if err != nil {
			return nil, err
		}
		answers[question.Key] = answer
	}
	return answers, nil
}

func convertAnswer(q integration.Question, raw wireAnswer) (integration.Answer, error) {
	if raw.Type != string(q.Kind) {
		return integration.Answer{}, invalid("answer type does not match the question type")
	}
	switch q.Kind {
	case integration.KindNoul:
		return convertNoul(raw)
	case integration.KindChoice:
		return convertChoice(q, raw)
	case integration.KindScore:
		return convertScore(q, raw)
	default:
		return integration.Answer{}, invalid("question kind is not supported")
	}
}
