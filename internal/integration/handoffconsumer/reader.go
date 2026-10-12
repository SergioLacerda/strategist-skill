package handoffconsumer

import (
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
)

// Reader reads the projected fields of a boundary artifact through the same
// catalog the pre-check projects from, so a calibration case contains exactly
// the fields a handoff would offer the provider and nothing else.
type Reader struct{}

// Package reads the package fields from explicit task and analysis files.
func (Reader) Package(tasksPath, analysisPath string) (calibration.Fields, error) {
	source, err := NewPackageSourceFiles(tasksPath, analysisPath)
	if err != nil {
		return nil, err
	}
	return collect(TransitionArchivistToGate, source), nil
}

// Ranger reads the fields of a Ranger analysis artifact.
func (Reader) Ranger(path string) (calibration.Fields, error) {
	source, err := NewRangerSource(path)
	if err != nil {
		return nil, err
	}
	return collect(handoff.TransitionRangerToArchivist, source), nil
}

func collect(transition string, source Source) calibration.Fields {
	fields := calibration.Fields{}
	for _, name := range Fields(transition) {
		if value, ok := source.Field(name); ok && value != "" {
			fields[name] = value
		}
	}
	return fields
}
