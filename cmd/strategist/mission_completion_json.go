package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

func readMissionCompletion(cmd *cobra.Command) (domain.MissionInvocationCompletion, error) {
	var raw json.RawMessage
	decoder := json.NewDecoder(cmd.InOrStdin())
	if err := decoder.Decode(&raw); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: more than one object was supplied")
	} else if err != io.EOF {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: trailing data: %w", err)
	}
	if err := validateCompletionObject(raw); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	var completion domain.MissionInvocationCompletion
	strict := json.NewDecoder(bytes.NewReader(raw))
	if err := strict.Decode(&completion); err != nil {
		return domain.MissionInvocationCompletion{}, fmt.Errorf("read completion JSON: %w", err)
	}
	return completion, nil
}

// validateCompletionObject rejects duplicate top-level fields before decoding
// into the completion struct. Unknown fields remain ignored intentionally:
// host-controlled metadata is untrusted and must not become authority, while
// rejecting it would make harmless provider annotations a compatibility break.
func validateCompletionObject(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	first, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("completion must be one JSON object: %w", err)
	}
	delim, ok := first.(json.Delim)
	if !ok || delim != '{' {
		return fmt.Errorf("completion must be one JSON object")
	}
	if err := validateCompletionFields(decoder); err != nil {
		return err
	}
	return validateCompletionEnd(decoder)
}

func validateCompletionFields(decoder *json.Decoder) error {
	seen := map[string]bool{}
	for decoder.More() {
		name, err := readCompletionField(decoder, seen)
		if err != nil {
			return err
		}
		seen[name] = true
	}
	return nil
}

func readCompletionField(decoder *json.Decoder, seen map[string]bool) (string, error) {
	key, err := decoder.Token()
	if err != nil {
		return "", fmt.Errorf("completion object is malformed: %w", err)
	}
	name, ok := key.(string)
	if !ok {
		return "", fmt.Errorf("completion object field name is not a string")
	}
	if seen[name] {
		return "", fmt.Errorf("completion object contains duplicate field %q", name)
	}
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("completion object field %q has invalid value: %w", name, err)
	}
	return name, nil
}

func validateCompletionEnd(decoder *json.Decoder) error {
	last, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("completion object is unclosed: %w", err)
	}
	if delim, ok := last.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("completion object is unclosed")
	}
	return nil
}
