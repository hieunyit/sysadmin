package internal

import (
	"strings"
	"testing"
)

func TestAPIErrorIncludesDetailsAndRequestID(t *testing.T) {
	t.Parallel()

	err := (&APIError{
		Method:    "POST",
		Path:      "/api/v1/keycloak/users",
		Status:    400,
		Code:      "invalid_argument",
		Message:   "validation failed",
		RequestID: "req-123",
		Details: map[string]string{
			"last_name": "is required",
			"email":     "must be a valid email address",
		},
	}).Error()

	if !strings.Contains(err, "validation failed") {
		t.Fatalf("expected message, got %q", err)
	}
	if !strings.Contains(err, "Details:") {
		t.Fatalf("expected details section, got %q", err)
	}
	if !strings.Contains(err, "Request ID: req-123") {
		t.Fatalf("expected request id, got %q", err)
	}
}

func TestAPIErrorFriendlyMessageStillIncludesRequestID(t *testing.T) {
	t.Parallel()

	err := (&APIError{
		Method:    "POST",
		Path:      "/api/v1/openvpn/access-lists:append",
		Status:    412,
		Code:      "precondition_failed",
		Message:   `group "Test" has no OpenVPN ruleset for domain routing`,
		RequestID: "req-456",
	}).Error()

	if !strings.Contains(err, "How to fix:") {
		t.Fatalf("expected friendly message, got %q", err)
	}
	if !strings.Contains(err, "Request ID: req-456") {
		t.Fatalf("expected request id, got %q", err)
	}
}
