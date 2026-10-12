package hardening

import "testing"

func TestDefaultAuthorityMapIsUniqueAndExplicit(t *testing.T) {
	mapValue := DefaultAuthorityMap()
	if err := mapValue.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(mapValue) < 5 {
		t.Fatalf("authority map too small: %d", len(mapValue))
	}
}

func TestAuthorityMapRejectsDuplicateFacts(t *testing.T) {
	if err := (AuthorityMap{{Fact: "x", Owner: "a"}, {Fact: "x", Owner: "b"}}).Validate(); err == nil {
		t.Fatal("expected duplicate fact error")
	}
}
