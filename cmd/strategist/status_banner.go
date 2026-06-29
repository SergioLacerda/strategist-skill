package main

import "fmt"

const bannerWidth = 69

// printStatusBanner prints the ASCII header used by human-status commands.
func printStatusBanner(command string) {
	line := fmt.Sprintf("| STRATEGIST :: %-52s|", command)
	border := "+" + repeatChar('-', bannerWidth) + "+"
	fmt.Println()
	fmt.Println(" ", border)
	fmt.Println(" ", line)
	fmt.Println(" ", border)
	fmt.Println()
}

func repeatChar(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}
