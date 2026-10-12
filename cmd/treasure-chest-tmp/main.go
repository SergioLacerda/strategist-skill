// Package main exposes the temporary Treasure Chest Weapon command.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/weapon/treasuretmp"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("treasure-chest-tmp: command is required (search, status, or prepare)")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "workspace root")
	jsonOutput := fs.Bool("json", false, "emit JSON")
	intent := fs.String("intent", "", "retrieval intent")
	concepts := stringArrayFlag{}
	symbols := stringArrayFlag{}
	fs.Var(&concepts, "concept", "retrieval concept")
	fs.Var(&symbols, "symbol", "retrieval symbol")
	limit := fs.Int("token-budget", 20, "maximum number of items")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("treasure-chest-tmp: parse flags: %w", err)
	}
	if !*jsonOutput {
		return fmt.Errorf("treasure-chest-tmp: --json is required")
	}
	return runCommand(args[0], stdout, *root, *intent, concepts, symbols, *limit)
}

func runCommand(command string, stdout io.Writer, root, intent string, concepts, symbols []string, limit int) error {
	encoder := json.NewEncoder(stdout)
	switch command {
	case "status":
		return encodeStatus(encoder, root)
	case "prepare":
		return encodePrepare(encoder, root)
	case "search":
		return encodeSearch(encoder, root, intent, concepts, symbols, limit)
	default:
		return fmt.Errorf("treasure-chest-tmp: unsupported command %q", command)
	}
}

func encodeStatus(encoder *json.Encoder, root string) error {
	if err := encoder.Encode(treasuretmp.Status(root)); err != nil {
		return fmt.Errorf("treasure-chest-tmp: encode status: %w", err)
	}
	return nil
}

func encodePrepare(encoder *json.Encoder, root string) error {
	response, err := treasuretmp.Prepare(root)
	if err != nil {
		return fmt.Errorf("treasure-chest-tmp: prepare: %w", err)
	}
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("treasure-chest-tmp: encode prepare response: %w", err)
	}
	return nil
}

func encodeSearch(encoder *json.Encoder, root, intent string, concepts, symbols []string, limit int) error {
	response, err := treasuretmp.Search(treasuretmp.Request{Root: root, Intent: intent, Concepts: concepts, Symbols: symbols, TokenBudget: limit})
	if err != nil {
		return fmt.Errorf("treasure-chest-tmp: search: %w", err)
	}
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("treasure-chest-tmp: encode search response: %w", err)
	}
	return nil
}

type stringArrayFlag []string

func (f *stringArrayFlag) String() string         { return fmt.Sprint([]string(*f)) }
func (f *stringArrayFlag) Set(value string) error { *f = append(*f, value); return nil }
