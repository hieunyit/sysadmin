package application

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"backend/internal/domain/services"
)

func TestOpenVPNAdminServiceListUsersSanitizesProfiles(t *testing.T) {
	t.Parallel()

	openvpn := &fakeOpenVPNPolicyService{
		listUsersFn: func(context.Context, services.OpenVPNUserListQuery) (map[string]any, error) {
			return map[string]any{
				"profiles": []any{
					map[string]any{
						"name":             "alice",
						"group":            "vpn-users",
						"password_defined": "true",
						"mfa_status":       "disabled",
						"admin":            true,
					},
				},
			}, nil
		},
	}

	svc := NewOpenVPNAdminService(openvpn, nil)
	out, err := svc.ListUsers(context.Background(), services.OpenVPNUserListQuery{Limit: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profiles, err := extractProfiles(out)
	if err != nil {
		t.Fatalf("unexpected extract error: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if got := formatMapString(profiles[0], "name"); got != "alice" {
		t.Fatalf("unexpected username: %q", got)
	}
	if _, ok := profiles[0]["admin"]; ok {
		t.Fatalf("expected admin to be removed from profile")
	}
}

func TestOpenVPNAdminServiceApplyAccessEntriesNoCrossSystemNotification(t *testing.T) {
	t.Parallel()

	openvpn := &fakeOpenVPNPolicyService{
		appendAccessListFn: func(_ context.Context, items []services.AccessRouteItem) error {
			if len(items) != 1 {
				t.Fatalf("expected 1 access item, got %d", len(items))
			}
			return nil
		},
	}
	mailer := &fakeMailTemplateSender{enabled: true}
	notifications := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")

	svc := NewOpenVPNAdminService(openvpn, notifications)
	username := "alice"
	target := "10.0.0.0/8"
	if err := svc.ApplyAccessEntries(context.Background(), "append", []services.OpenVPNAccessEntryInput{{
		Username: &username,
		Target:   &target,
	}}, "api-admin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mailer.calls) != 0 {
		t.Fatalf("expected 0 notification mails, got %d", len(mailer.calls))
	}
}
