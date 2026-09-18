package conformance

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// DecodeMatrix parses and validates a matrix fixture.
func DecodeMatrix(data []byte) (Matrix, error) {
	var matrix Matrix
	if err := yaml.Unmarshal(data, &matrix); err != nil {
		return Matrix{}, fmt.Errorf("conformance matrix: decode: %w", err)
	}
	if err := matrix.Validate(); err != nil {
		return Matrix{}, err
	}
	return matrix, nil
}
