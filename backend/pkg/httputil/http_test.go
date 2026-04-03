package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainerr "backend/internal/domain/errors"
)

func TestDecodeJSONRejectsUnknownField(t *testing.T) {
	t.Parallel()

	var payload struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice","extra":"x"}`))
	err := DecodeJSON(req, &payload)
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
	if got := de.Details["extra"]; got != "is not allowed" {
		t.Fatalf("expected unknown field detail, got %#v", de.Details)
	}
}

func TestDecodeJSONRejectsMultipleObjects(t *testing.T) {
	t.Parallel()

	var payload struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"}{"name":"bob"}`))
	err := DecodeJSON(req, &payload)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	de, ok := err.(*domainerr.DomainError)
	if !ok {
		t.Fatalf("expected domain error, got %T", err)
	}
	if got := de.Details["body"]; got != "must contain a single JSON object" {
		t.Fatalf("expected single object detail, got %#v", de.Details)
	}
}

func TestWriteErrorIncludesDetailsAndRequestID(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	err := domainerr.NewWithDetails(domainerr.CodeInvalidArgument, "validation failed", map[string]string{
		"userExpiryVPN": "must be a valid date in dd/MM/yyyy",
	})
	WriteError(rec, err, "req-123")

	body := rec.Body.String()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(body, `"request_id":"req-123"`) {
		t.Fatalf("expected request_id in body, got %s", body)
	}
	if !strings.Contains(body, `"userExpiryVPN":"must be a valid date in dd/MM/yyyy"`) {
		t.Fatalf("expected details in body, got %s", body)
	}
}

func TestDecodeJSONRejectsBodyTooLarge(t *testing.T) {
	t.Parallel()

	var payload struct {
		Name string `json:"name"`
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice"}`))
	req.Body = http.MaxBytesReader(rec, req.Body, 8)

	err := DecodeJSON(req, &payload)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	de, ok := err.(*domainerr.DomainError)
	if !ok {
		t.Fatalf("expected domain error, got %T", err)
	}
	if got := de.Details["body"]; got != "must not exceed 8 bytes" {
		t.Fatalf("expected body-too-large detail, got %#v", de.Details)
	}
}

func TestWriteErrorDoesNotLeakExternalFailureCause(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	err := domainerr.Wrap(domainerr.CodeExternalFailure, "keycloak create user failed", errors.New("status=500 body={\"detail\":\"secret\"}"))
	WriteError(rec, err, "req-456")

	body := rec.Body.String()
	if strings.Contains(body, "secret") {
		t.Fatalf("expected external failure cause to be sanitized, got %s", body)
	}
	if !strings.Contains(body, `"request_id":"req-456"`) {
		t.Fatalf("expected request_id in body, got %s", body)
	}
}
