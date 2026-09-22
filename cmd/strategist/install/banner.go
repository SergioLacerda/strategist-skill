package install

import "fmt"

func printCompleteBanner(target string, wizard, partial bool) {
	mode := "silent"
	if wizard {
		mode = "wizard"
	}
	fmt.Println()
	if partial {
		printPartialBanner(target, mode)
		return
	}
	printFullBanner(target, mode)
}
func printPartialBanner(target, mode string) {
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
func printFullBanner(target, mode string) {
	fmt.Println("  ┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("  │  STRATEGIST  ◆  install complete                                    │")
	fmt.Println("  └─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("     target  %s\n", target)
	fmt.Printf("     mode    %s\n", mode)
	fmt.Println()
}
