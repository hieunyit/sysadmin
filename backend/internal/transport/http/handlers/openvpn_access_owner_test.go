package handlers

import (
	"testing"

	domainerr "backend/internal/domain/errors"
)

func TestMapAccessEntriesForOwnerUserInjectsUsername(t *testing.T) {
	t.Parallel()

	items := []accessRouteDTO{
		{Target: strPtr("10.0.0.0/8")},
	}

	out, err := mapAccessEntriesForOwner(items, "user", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out))
	}
	if out[0].Username == nil || *out[0].Username != "alice" {
		t.Fatalf("expected injected username owner, got %#v", out[0].Username)
	}
	if out[0].Groupname != nil {
		t.Fatalf("expected groupname to be nil for user owner")
	}
}

func TestMapAccessEntriesForOwnerUserRejectsGroupname(t *testing.T) {
	t.Parallel()

	items := []accessRouteDTO{
		{
			Target:    strPtr("10.0.0.0/8"),
			Groupname: strPtr("ops"),
		},
	}

	_, err := mapAccessEntriesForOwner(items, "user", "alice")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if de, ok := err.(*domainerr.DomainError); !ok || de.Code != domainerr.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument domain error, got %T (%v)", err, err)
	}
}

func strPtr(v string) *string {
	return &v
}
