package passwordutil

import (
	"strings"
	"testing"
)

func TestGenerateTemporary(t *testing.T) {
	t.Parallel()

	password, err := GenerateTemporary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(password) != 14 {
		t.Fatalf("expected password length 14, got %d", len(password))
	}

	assertContainsAny := func(label, charset string) {
		t.Helper()
		if !strings.ContainsAny(password, charset) {
			t.Fatalf("expected generated password to contain %s, got %q", label, password)
		}
	}

	assertContainsAny("lowercase", lowerCharset)
	assertContainsAny("uppercase", upperCharset)
	assertContainsAny("digit", digitCharset)
	assertContainsAny("symbol", symbolCharset)
}
