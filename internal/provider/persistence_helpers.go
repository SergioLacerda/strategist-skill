package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func validateProviderID(id string) error {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, `/\\`) {
		return fmt.Errorf("provider id invalid: %q", id)
	}
	return nil
}

func transactionID(instanceID, slot string) string {
	return "provider-add-" + slot + "-" + instanceID
}

func newLockBindingGeneration(lock domain.PluginLockFile, slot string) int64 {
	return nextGeneration(lock, slot)
}

func bindingGeneration(lock domain.PluginLockFile, slot string) int64 {
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			return binding.Generation
		}
	}
	return 0
}

func compileWorkspace(root string) error {
	if _, err := os.Stat(filepath.Join(root, "active.yaml")); err != nil {
		return fmt.Errorf("compile prerequisite active.yaml: %w", err)
	}
	if err := (compile.Compiler{}).CompileAll(root, filepath.Join(root, "knowledge.index.yaml")); err != nil {
		return fmt.Errorf("compile provider runtime: %w", err)
	}
	return nil
}
