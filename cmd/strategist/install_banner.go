package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// installIsPartial reports whether the just-completed install has an
// uncompiled/partial runtime — i.e. CompileAll's warning-only failure path
// was taken and no manifest was produced. Only meaningful immediately after a
// successful Service.Install call.
func installIsPartial(target string) bool {
	manifestPath := filepath.Join(target, ".strategist", ".compiled", ".manifest.gz")
	_, err := os.Stat(manifestPath)
	return err != nil
}

func printInstallCompleteBanner(target string, wizard, partial bool) {
	mode := "silent"
	if wizard {
		mode = "wizard"
	}
	fmt.Println()
	if partial {
		printPartialInstallBanner(target, mode)
		return
	}
	printFullInstallBanner(target, mode)
}

func printPartialInstallBanner(target, mode string) {
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  STRATEGIST  ◆  install complete (partial — compile warning)        │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("     target  %s\n", target)
	fmt.Printf("     mode    %s\n", mode)
	fmt.Println()
	fmt.Println("     ⚠ compile failed during install — runtime is uncompiled/partial.")
	fmt.Println("       Run: strategist compile   (or re-run install with --strict-compile")
	fmt.Println("       to make this fatal instead of warning-only next time)")
	fmt.Println()
}

func printFullInstallBanner(target, mode string) {
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  STRATEGIST  ◆  install complete                                    │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("     target  %s\n", target)
	fmt.Printf("     mode    %s\n", mode)
	fmt.Println()
}
