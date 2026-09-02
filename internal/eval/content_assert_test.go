package eval

import "testing"

func TestCheckContains(t *testing.T) {
	if v := checkContains("hello world", Assertion{Type: AssertContains, Value: "world"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	v := checkContains("hello world", Assertion{Type: AssertContains, Value: "xyz"})
	if v == nil {
		t.Fatal("expected a violation, got nil")
		return
	}
	if v.Expected != "xyz" {
		t.Fatalf("unexpected Expected: %q", v.Expected)
	}
}

func TestCheckNotContains(t *testing.T) {
	if v := checkNotContains("hello world", Assertion{Type: AssertNotContains, Value: "xyz"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	v := checkNotContains("hello world", Assertion{Type: AssertNotContains, Value: "world"})
	if v == nil {
		t.Fatal("expected a violation, got nil")
	}
}

func TestCheckRegex(t *testing.T) {
	if v := checkRegex("hello world", Assertion{Type: AssertRegex, Value: `wor.d`}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	if v := checkRegex("hello world", Assertion{Type: AssertRegex, Value: `xyz\d+`}); v == nil {
		t.Fatal("expected a violation for a non-matching pattern, got nil")
	}
}

func TestCheckRegex_InvalidPattern(t *testing.T) {
	v := checkRegex("hello world", Assertion{Type: AssertRegex, Value: "["})
	if v == nil {
		t.Fatal("expected a violation for an invalid regex, got nil")
		return
	}
	if v.Message != "invalid regex pattern" {
		t.Fatalf("unexpected message: %q", v.Message)
	}
}

func TestCheckMaxTokens(t *testing.T) {
	if v := checkMaxTokens("one two three", Assertion{Type: AssertMaxTokens, Value: "5"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	if v := checkMaxTokens("one two three", Assertion{Type: AssertMaxTokens, Value: "2"}); v == nil {
		t.Fatal("expected a violation when the token budget is exceeded, got nil")
	}
}

func TestCheckMaxTokens_NotAnInteger(t *testing.T) {
	v := checkMaxTokens("one two three", Assertion{Type: AssertMaxTokens, Value: "not-a-number"})
	if v == nil {
		t.Fatal("expected a violation for a non-integer budget, got nil")
		return
	}
	if v.Message != "max-tokens value is not an integer" {
		t.Fatalf("unexpected message: %q", v.Message)
	}
}

func TestCheckForbiddenToolCall(t *testing.T) {
	if v := checkForbiddenToolCall("run npm test", Assertion{Type: AssertForbiddenToolCall, Value: "git commit"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	if v := checkForbiddenToolCall("run git commit -m x", Assertion{Type: AssertForbiddenToolCall, Value: "git commit"}); v == nil {
		t.Fatal("expected a violation when the forbidden call is present, got nil")
	}
}

func TestCheckRequiredSections(t *testing.T) {
	content := "# Summary\ncontent\n## Details\nmore\n"
	if got := checkRequiredSections(content, Assertion{Type: AssertRequiredSections, Value: "Summary, Details"}); len(got) != 0 {
		t.Fatalf("expected no violations, got %+v", got)
	}
	got := checkRequiredSections(content, Assertion{Type: AssertRequiredSections, Value: "Summary, Missing"})
	if len(got) != 1 {
		t.Fatalf("expected 1 violation for the missing section, got %d: %+v", len(got), got)
	}
	if got[0].Expected != "Missing" {
		t.Fatalf("unexpected Expected: %q", got[0].Expected)
	}
}

func TestCheckSourceCitations(t *testing.T) {
	content := "see docs/a.md and internal/b.go for details"
	if v := checkSourceCitations(content, Assertion{Type: AssertSourceCitations, Value: "2"}); v != nil {
		t.Fatalf("expected no violation, got %+v", v)
	}
	if v := checkSourceCitations(content, Assertion{Type: AssertSourceCitations, Value: "5"}); v == nil {
		t.Fatal("expected a violation when there are fewer citations than required, got nil")
	}
}

func TestCheckSourceCitations_NotAnInteger(t *testing.T) {
	v := checkSourceCitations("docs/a.md", Assertion{Type: AssertSourceCitations, Value: "not-a-number"})
	if v == nil {
		t.Fatal("expected a violation for a non-integer minimum, got nil")
		return
	}
	if v.Message != "source-citations value is not an integer" {
		t.Fatalf("unexpected message: %q", v.Message)
	}
}

func TestEvaluateContentAssertions(t *testing.T) {
	res := &ScenarioResult{}
	assertions := []Assertion{
		{Type: AssertContains, Value: "hello"},
		{Type: AssertNotContains, Value: "forbidden"},
		{Type: AssertRegex, Value: "wor.d"},
		{Type: AssertMaxTokens, Value: "10"},
		{Type: AssertForbiddenToolCall, Value: "rm -rf"},
		{Type: AssertRequiredSections, Value: "Summary"},
		{Type: AssertSourceCitations, Value: "1"},
	}
	evaluateContentAssertions("# Summary\nhello world docs/a.md\n", assertions, res)
	if len(res.Violations) != 0 {
		t.Fatalf("expected no violations, got %+v", res.Violations)
	}

	res2 := &ScenarioResult{}
	evaluateContentAssertions("nothing matches here", []Assertion{{Type: AssertContains, Value: "missing"}}, res2)
	if len(res2.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(res2.Violations))
	}
}
