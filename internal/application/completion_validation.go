package application

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ValidateCompletionObject rejects duplicate top-level fields before the raw
// host response is decoded into the trusted completion shape. Unknown fields
// remain compatible and are intentionally ignored.
func ValidateCompletionObject(raw []byte) error {
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
