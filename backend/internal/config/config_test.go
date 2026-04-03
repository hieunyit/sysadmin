package config

import (
	"strings"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("KEYCLOAK_BASE_URL", "https://admin-sso.mobifonesolutions.vn")
	t.Setenv("KEYCLOAK_REALM", "mbfs-solutions")
	t.Setenv("KEYCLOAK_CLIENT_ID", "client-id")
	t.Setenv("KEYCLOAK_CLIENT_SECRET", "client-secret")
	t.Setenv("OPENVPN_BASE_URL", "https://vpn.example.com/api")
	t.Setenv("OPENVPN_USERNAME", "openvpn")
	t.Setenv("OPENVPN_PASSWORD", "secret")
	t.Setenv("ADMIN_API_TOKEN", "token")
}

func TestLoadRejectsInvalidBool(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("OPENVPN_USE_SHADOW_OBJECTS", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid bool for OPENVPN_USE_SHADOW_OBJECTS") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid duration for HTTP_READ_TIMEOUT") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSetsDefaultsForShadowObjectsAndComponentLockFile(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.OpenVPN.UseShadowObjects {
		t.Fatalf("expected UseShadowObjects default true")
	}
	if got := cfg.Keycloak.ComponentLockFile; got != "/tmp/backend-keycloak-component.lock" {
		t.Fatalf("unexpected component lock file: %q", got)
	}
}

func TestLoadRequiresSMTPFieldsWhenEnabled(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SMTP_ENABLED", "true")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required env: SMTP_HOST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRequiresLDAPSmtpFieldsWhenEnabled(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SMTP_LDAP_ENABLED", "true")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required env: SMTP_LDAP_HOST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadLDAPSmtpInheritsPrimarySMTPSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SMTP_ENABLED", "true")
	t.Setenv("SMTP_HOST", "smtpdm-ap-southeast-1.aliyun.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_TLS_MODE", "direct")
	t.Setenv("SMTP_FROM_ADDRESS", "noreply@mbfs.vn")
	t.Setenv("SMTP_USERNAME", "primary")
	t.Setenv("SMTP_PASSWORD", "primary-secret")
	t.Setenv("SMTP_LDAP_ENABLED", "true")
	t.Setenv("SMTP_LDAP_USERNAME", "ldap-user")
	t.Setenv("SMTP_LDAP_PASSWORD", "ldap-secret")
	t.Setenv("SMTP_LDAP_FROM_ADDRESS", "ldap-noreply@mbfs.vn")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg.SMTPLDAP.Host; got != "smtpdm-ap-southeast-1.aliyun.com" {
		t.Fatalf("unexpected inherited host: %q", got)
	}
	if got := cfg.SMTPLDAP.Port; got != 465 {
		t.Fatalf("unexpected inherited port: %d", got)
	}
	if got := cfg.SMTPLDAP.TLSMode; got != "direct" {
		t.Fatalf("unexpected inherited tls mode: %q", got)
	}
	if got := cfg.SMTPLDAP.Username; got != "ldap-user" {
		t.Fatalf("unexpected ldap username: %q", got)
	}
	if got := cfg.SMTPLDAP.FromAddress; got != "ldap-noreply@mbfs.vn" {
		t.Fatalf("unexpected ldap from address: %q", got)
	}
}
