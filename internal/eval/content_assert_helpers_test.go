package eval

import "testing"

func TestHasHeading(t *testing.T) {
	content := "intro text\n# Summary\nbody\n## Nested Details\nmore\n"
	if !hasHeading(content, "Summary") {
		t.Fatal("expected to find the Summary heading")
	}
	if !hasHeading(content, "Nested Details") {
		t.Fatal("expected to find the Nested Details heading")
	}
	if hasHeading(content, "Missing") {
		t.Fatal("did not expect to find a Missing heading")
	}
	if hasHeading("Summary appears only in body text, not as a heading", "Summary") {
		t.Fatal("a non-heading line containing the text must not count")
	}
}

func TestCountSourceCitations(t *testing.T) {
	content := "see docs/a.md and internal/b.go, also docs/a.md again"
	if got := countSourceCitations(content); got != 2 {
		t.Fatalf("expected 2 distinct citations (duplicate deduped), got %d", got)
	}
	if got := countSourceCitations("no citations here"); got != 0 {
		t.Fatalf("expected 0 citations, got %d", got)
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a, b ,, c")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if got := splitCSV(""); len(got) != 0 {
		t.Fatalf("expected empty slice for empty input, got %v", got)
	}
}

func TestTruncate(t *testing.T) {
	short := "short string"
	if got := truncate(short); got != short {
		t.Fatalf("expected unchanged short string, got %q", got)
	}
	long := ""
	for i := 0; i < 200; i++ {
		long += "x"
	}
	got := truncate(long)
	if len(got) != 123 { // 120 chars + "..."
		t.Fatalf("expected truncated length 123, got %d", len(got))
	}
	if got[120:] != "..." {
		t.Fatalf("expected truncation suffix '...', got %q", got[120:])
	}
}
