package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsNonJSONOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run([]string{"status", "--root", "/tmp"}, &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "--json is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run([]string{"local", "--json"}, &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "unsupported command") {
		t.Fatalf("unexpected error: %v", err)
	}
}
