// Package domain defines core types, errors, and ports for the Strategist skill.
package domain

import "errors"

var (
	// ErrArtifactAbsent is returned when the compiled artifact file does not exist.
	ErrArtifactAbsent = errors.New("artifact does not exist")

	// ErrManifestMissing is returned when no .manifest.gz accompanies the artifact.
	ErrManifestMissing = errors.New("manifest not found")

	// ErrSourceStale is returned when a source file was modified after the artifact was compiled.
	ErrSourceStale = errors.New("source file modified after artifact")
)
