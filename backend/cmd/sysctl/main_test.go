package main

import (
	"encoding/json"
	"testing"
)

func TestValidateIdentitySourceCreateFlagsLocal(t *testing.T) {
	t.Parallel()

	err := validateIdentitySourceCreateFlags("local", "", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "create-local requires --user-type, --company-name" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestValidateIdentitySourceCreateFlagsLDAP(t *testing.T) {
	t.Parallel()

	err := validateIdentitySourceCreateFlags("ldap", "partner", "TEST", "", "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "create-ldap requires --phone, --department, --manager" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestValidateIdentitySourceCreateFlagsUserTypeRules(t *testing.T) {
	t.Parallel()

	if err := validateIdentitySourceCreateFlags("local", "employee", "TEST", "", "", "", ""); err == nil {
		t.Fatal("expected local userType error, got nil")
	}
	if err := validateIdentitySourceCreateFlags("ldap", "partner", "", "", "0987654321", "Ops", "CN=Manager"); err == nil {
		t.Fatal("expected ldap userType error, got nil")
	}
}

func TestValidateIdentitySourceCreateFlagsValidatesOnboardDateFormat(t *testing.T) {
	t.Parallel()

	if err := validateIdentitySourceCreateFlags("local", "partner", "TEST", "20/04/2026", "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateIdentitySourceCreateFlags("local", "partner", "TEST", "2026-04-20", "", "", ""); err == nil {
		t.Fatal("expected invalid onboard date format error, got nil")
	}
}

func TestNormalizeOpenVPNUserProfilesDropsLegacyCrossFields(t *testing.T) {
	t.Parallel()

	items := []any{
		map[string]any{
			"name":              "alice",
			"group":             "vpn-users",
			"auth_method":       map[string]any{"value": "local"},
			"deny":              map[string]any{"value": "false"},
			"password_defined":  "true",
			"mfa_status":        "disabled",
			"vpn_expire_at":     "20/04/2026",
			"last_vpn_login_at": "2026-02-04T10:10:33Z",
			"admin":             true,
		},
	}

	out := normalizeOpenVPNUserProfiles(items)
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	row, ok := out[0].(map[string]any)
	if !ok {
		t.Fatalf("expected normalized row map, got %T", out[0])
	}
	if got := row["username"]; got != "alice" {
		t.Fatalf("expected username to be mapped, got %#v", got)
	}
	if got := row["auth_method"]; got != "local" {
		t.Fatalf("expected auth_method to be mapped, got %#v", got)
	}
	if got := row["deny"]; got != "false" {
		t.Fatalf("expected deny to be mapped, got %#v", got)
	}
	if _, ok := row["vpn_expire_at"]; ok {
		t.Fatalf("expected vpn_expire_at to be removed")
	}
	if _, ok := row["last_vpn_login_at"]; ok {
		t.Fatalf("expected last_vpn_login_at to be removed")
	}
	if _, ok := row["admin"]; ok {
		t.Fatalf("expected admin to be omitted from normalized row")
	}
}

func TestBuildUserImportPayloadIncludesNotificationEmail(t *testing.T) {
	t.Parallel()

	payload, err := buildUserImportPayload(map[string]string{
		normalizeCSVHeader("username"):           "ldap.user",
		normalizeCSVHeader("email"):              "ldap.user@mbfs.vn",
		normalizeCSVHeader("notification_email"): "notify@mbfs.vn",
		normalizeCSVHeader("first_name"):         "LDAP",
		normalizeCSVHeader("last_name"):          "User",
		normalizeCSVHeader("display_name"):       "LDAP User",
		normalizeCSVHeader("password"):           "Mbfs@111",
		normalizeCSVHeader("identity_source"):    "ldap",
		normalizeCSVHeader("enabled"):            "true",
		normalizeCSVHeader("email_verified"):     "true",
		normalizeCSVHeader("fullName"):           "LDAP User",
		normalizeCSVHeader("phone"):              "0987654321",
		normalizeCSVHeader("department"):         "Phòng Vận hành",
		normalizeCSVHeader("manager"):            "CN=Manager,OU=Users,DC=mbfs,DC=local",
		normalizeCSVHeader("onboardDate"):        "15/04/2026",
		normalizeCSVHeader("workAddress"):        "Tòa nhà MobiFone, Hà Nội",
		normalizeCSVHeader("userType"):           "employee",
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if got := formatCell(out["notification_email"]); got != "notify@mbfs.vn" {
		t.Fatalf("unexpected notification_email: %q", got)
	}
	attrs, ok := out["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes map, got %T", out["attributes"])
	}
	if got := formatCell(attrs["onboardDate"]); got != "15/04/2026" {
		t.Fatalf("unexpected onboardDate: %q", got)
	}
	if got := formatCell(attrs["workAddress"]); got != "Tòa nhà MobiFone, Hà Nội" {
		t.Fatalf("unexpected workAddress: %q", got)
	}
}

func TestBuildUserImportPayloadAllowsEmptyPassword(t *testing.T) {
	t.Parallel()

	payload, err := buildUserImportPayload(map[string]string{
		normalizeCSVHeader("username"):        "local.user",
		normalizeCSVHeader("email"):           "local.user@mbfs.vn",
		normalizeCSVHeader("first_name"):      "Local",
		normalizeCSVHeader("last_name"):       "User",
		normalizeCSVHeader("identity_source"): "local",
		normalizeCSVHeader("enabled"):         "true",
		normalizeCSVHeader("email_verified"):  "true",
		normalizeCSVHeader("fullName"):        "Local User",
		normalizeCSVHeader("userType"):        "partner",
		normalizeCSVHeader("companyName"):     "TEST",
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := formatCell(payload["password"]); got != "" {
		t.Fatalf("expected empty password to be preserved for backend generation, got %q", got)
	}
}

func TestApplyAccessListOwnerToPayloadForUser(t *testing.T) {
	t.Parallel()

	body := map[string]any{
		"items": []any{
			map[string]any{
				"target": "10.0.0.0/8",
			},
		},
	}

	if err := applyAccessListOwnerToPayload(body, "user", "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := body["items"].([]any)
	item := items[0].(map[string]any)
	if got := formatCell(item["username"]); got != "alice" {
		t.Fatalf("expected username owner to be injected, got %q", got)
	}
	if _, exists := item["groupname"]; exists {
		t.Fatalf("expected groupname to be removed for user owner")
	}
}

func TestApplyAccessListOwnerToPayloadRejectsMismatchedOwner(t *testing.T) {
	t.Parallel()

	body := map[string]any{
		"items": []any{
			map[string]any{
				"target":   "10.0.0.0/8",
				"username": "bob",
			},
		},
	}

	err := applyAccessListOwnerToPayload(body, "user", "alice")
	if err == nil {
		t.Fatal("expected owner mismatch error, got nil")
	}
}

func TestOpenVPNCommandScopesAccessListByOwner(t *testing.T) {
	t.Parallel()

	cmd := openvpnCmd(&rootOptions{})
	userCmd, _, err := cmd.Find([]string{"user", "access-list", "list"})
	if err != nil {
		t.Fatalf("expected user access-list list command, got error: %v", err)
	}
	if userCmd == nil || userCmd.Name() != "list" {
		t.Fatalf("unexpected user access-list command resolution: %#v", userCmd)
	}

	groupCmd, _, err := cmd.Find([]string{"group", "access-list", "append"})
	if err != nil {
		t.Fatalf("expected group access-list append command, got error: %v", err)
	}
	if groupCmd == nil || groupCmd.Name() != "append" {
		t.Fatalf("unexpected group access-list command resolution: %#v", groupCmd)
	}
}
