package main

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/spf13/cobra"
)

var compileRoot string

var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile all skill artifacts from a .strategist/ root",
	RunE: func(_ *cobra.Command, _ []string) error {
		if compileRoot == "" {
			compileRoot = ".strategist"
		}

		indexPath := filepath.Join(compileRoot, "knowledge.index.yaml")
		c := compile.Compiler{}
		if err := c.CompileAll(compileRoot, indexPath); err != nil {
			return fmt.Errorf("compile: %w", err)
		}

		fmt.Printf("[Strategist] compile complete → %s/.compiled/\n", compileRoot)
		return nil
	},
}

func init() {
	compileCmd.Flags().StringVar(&compileRoot, "root", "", "path to .strategist/ root (default: .strategist)")
}
