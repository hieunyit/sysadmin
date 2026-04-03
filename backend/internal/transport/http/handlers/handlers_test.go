package handlers

import (
	"net/http/httptest"
	"testing"

	domainerr "backend/internal/domain/errors"
)

func TestParseLimitOffsetRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/keycloak/users?limit=abc", nil)
	_, _, err := parseLimitOffset(req, 50)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	de, ok := err.(*domainerr.DomainError)
	if !ok {
		t.Fatalf("expected domain error, got %T", err)
	}
	if de.Code != domainerr.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument, got %s", de.Code)
	}
	if got := de.Details["limit"]; got != "must be a positive integer" {
		t.Fatalf("unexpected limit detail: %#v", de.Details)
	}
}

func TestParseLimitOffsetRejectsTooLargeLimit(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/keycloak/users?limit=501", nil)
	_, _, err := parseLimitOffset(req, 50)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	de := err.(*domainerr.DomainError)
	if got := de.Details["limit"]; got != "must be less than or equal to 500" {
		t.Fatalf("unexpected limit detail: %#v", de.Details)
	}
}

func TestParseLimitOffsetRejectsNegativeOffset(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/keycloak/users?offset=-1", nil)
	_, _, err := parseLimitOffset(req, 50)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	de := err.(*domainerr.DomainError)
	if got := de.Details["offset"]; got != "must be greater than or equal to 0" {
		t.Fatalf("unexpected offset detail: %#v", de.Details)
	}
}

func TestParseOptionalBoolQueryRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/openvpn/groups?enumerate_members=yes", nil)
	_, err := parseOptionalBoolQuery(req, "enumerate_members", false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	de := err.(*domainerr.DomainError)
	if got := de.Details["enumerate_members"]; got != "must be true or false" {
		t.Fatalf("unexpected bool detail: %#v", de.Details)
	}
}
