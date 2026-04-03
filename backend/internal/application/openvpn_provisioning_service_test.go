package application

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"backend/internal/domain/services"
)

func TestOpenVPNProvisioningServiceCreateUserFromKeycloakByUsername(t *testing.T) {
	t.Parallel()

	keycloak := &fakeKeycloakIdentityService{
		searchUsersFn: func(context.Context, string, int, int) ([]services.KeycloakUser, error) {
			return []services.KeycloakUser{{
				ID:          "kc-1",
				Username:    "alice",
				Email:       "alice@example.com",
				DisplayName: "Alice",
				Enabled:     true,
			}}, nil
		},
	}
	ensuredUser := ""
	ensuredGroup := ""
	setGroupUser := ""
	setGroupName := ""
	openvpn := &fakeOpenVPNPolicyService{
		ensureUserFn: func(_ context.Context, username string) error {
			ensuredUser = username
			return nil
		},
		ensureGroupFn: func(_ context.Context, groupname string) error {
			ensuredGroup = groupname
			return nil
		},
		setUserGroupFn: func(_ context.Context, username, groupname string) error {
			setGroupUser = username
			setGroupName = groupname
			return nil
		},
	}
	mailer := &fakeMailTemplateSender{enabled: true}
	notifications := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")

	svc := NewOpenVPNProvisioningService(openvpn, keycloak, notifications)
	result, err := svc.CreateUserFromKeycloak(context.Background(), "", "alice", "partners-vpn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OpenVPNUser != "alice" {
		t.Fatalf("unexpected openvpn user: %q", result.OpenVPNUser)
	}
	if result.OpenVPNGroup != "partners-vpn" {
		t.Fatalf("unexpected openvpn group: %q", result.OpenVPNGroup)
	}
	if ensuredUser != "alice" {
		t.Fatalf("expected ensure user alice, got %q", ensuredUser)
	}
	if ensuredGroup != "partners-vpn" {
		t.Fatalf("expected ensure group partners-vpn, got %q", ensuredGroup)
	}
	if setGroupUser != "alice" || setGroupName != "partners-vpn" {
		t.Fatalf("unexpected group assignment %q %q", setGroupUser, setGroupName)
	}
	if len(mailer.calls) != 1 || mailer.calls[0].Template != "vpn_provisioned" {
		t.Fatalf("expected one vpn_provisioned notification, got %#v", mailer.calls)
	}
}

func TestOpenVPNProvisioningServiceCreateUserFromKeycloakRequiresIdentifier(t *testing.T) {
	t.Parallel()

	svc := NewOpenVPNProvisioningService(&fakeOpenVPNPolicyService{}, &fakeKeycloakIdentityService{}, nil)
	_, err := svc.CreateUserFromKeycloak(context.Background(), "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestOpenVPNProvisioningServiceCreateUserFromKeycloakRejectsDisabledUser(t *testing.T) {
	t.Parallel()

	keycloak := &fakeKeycloakIdentityService{
		getUserFn: func(context.Context, string) (services.KeycloakUser, error) {
			return services.KeycloakUser{ID: "kc-1", Username: "alice", Enabled: false}, nil
		},
	}
	svc := NewOpenVPNProvisioningService(&fakeOpenVPNPolicyService{}, keycloak, nil)
	_, err := svc.CreateUserFromKeycloak(context.Background(), "kc-1", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
