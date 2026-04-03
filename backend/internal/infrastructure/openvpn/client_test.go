package openvpn

import "testing"

func TestParseAccessServiceSpecRejectsOutOfRangePort(t *testing.T) {
	t.Parallel()

	if _, err := parseAccessServiceSpec("tcp:70000"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseAccessServiceSpecRejectsDescendingPortRange(t *testing.T) {
	t.Parallel()

	if _, err := parseAccessServiceSpec("tcp:443-80"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
