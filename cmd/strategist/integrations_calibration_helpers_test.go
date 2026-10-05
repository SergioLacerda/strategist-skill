package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

type cobraCmd = cobra.Command

func runCalibrationCommand(t *testing.T, fn func(*cobraCmd) error) (string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := fn(cmd)
	return out.String(), err
}

func removeCalibration(basePath, id string) error {
	return os.Remove(filepath.Join(basePath, "dojo", "calibrations", id+".json"))
}
