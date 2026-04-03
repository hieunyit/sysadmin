package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateIdentitySourceCreateFlagsLocal(t *testing.T) {
	t.Parallel()

	err := validateIdentitySourceCreateFlags("local", "", "", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "create-local requires --user-type, --company-name" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestValidateIdentitySourceCreateFlagsLDAP(t *testing.T) {
	t.Parallel()

	err := validateIdentitySourceCreateFlags("ldap", "partner", "TEST", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "create-ldap requires --phone, --department, --manager" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestValidateIdentitySourceCreateFlagsUserTypeRules(t *testing.T) {
	t.Parallel()

	if err := validateIdentitySourceCreateFlags("local", "employee", "TEST", "", "", "", "", ""); err == nil {
		t.Fatal("expected local userType error, got nil")
	}
	if err := validateIdentitySourceCreateFlags("ldap", "partner", "", "", "", "0987654321", "Ops", "CN=Manager"); err == nil {
		t.Fatal("expected ldap userType error, got nil")
	}
}

func TestValidateIdentitySourceCreateFlagsValidatesExpiryFormat(t *testing.T) {
	t.Parallel()

	if err := validateIdentitySourceCreateFlags("local", "partner", "TEST", "20/04/2026", "", "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateIdentitySourceCreateFlags("local", "partner", "TEST", "2026-04-20", "", "", "", ""); err == nil {
		t.Fatal("expected invalid expiry format error, got nil")
	}
}

func TestNormalizeOpenVPNUserProfilesIncludesVPNExpireAt(t *testing.T) {
	t.Parallel()

	items := []any{
		map[string]any{
			"name":              "alice",
			"group":             "vpn-users",
			"password_defined":  "true",
			"mfa_status":        "disabled",
			"vpn_expire_at":     "20/04/2026",
			"last_vpn_login_at": "2026-02-04T10:10:33Z",
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
	if got := row["vpn_expire_at"]; got != "20/04/2026" {
		t.Fatalf("expected vpn_expire_at to be preserved, got %#v", got)
	}
	if got := row["last_vpn_login_at"]; got != "2026-02-04T10:10:33Z" {
		t.Fatalf("expected last_vpn_login_at to be preserved, got %#v", got)
	}
	if _, ok := row["admin"]; ok {
		t.Fatalf("expected admin to be omitted from normalized row")
	}
}

func TestFormatTableCellFormatsLastVPNLoginAtForOperators(t *testing.T) {
	t.Parallel()

	got := formatTableCell("last_vpn_login_at", "2026-02-04T11:50:33+07:00")
	if got != "04/02/2026 11:50:33 +07" {
		t.Fatalf("unexpected formatted datetime: %q", got)
	}
}

func TestMatchesVPNInactiveDaysIncludesNeverLoggedIn(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 30, 10, 0, 0, 0, vpnFilterLocation())
	if !matchesVPNInactiveDays("", 30, now) {
		t.Fatal("expected empty last login to be treated as inactive")
	}
	if !matchesVPNInactiveDays("2026-02-20T10:00:00+07:00", 30, now) {
		t.Fatal("expected old login to match inactive filter")
	}
	if matchesVPNInactiveDays("2026-03-20T10:00:00+07:00", 30, now) {
		t.Fatal("expected recent login to not match inactive filter")
	}
}

func TestMatchesVPNExpiringInDays(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 30, 10, 0, 0, 0, vpnFilterLocation())
	if !matchesVPNExpiringInDays("30/03/2026", 7, now) {
		t.Fatal("expected today expiry to match")
	}
	if !matchesVPNExpiringInDays("05/04/2026", 7, now) {
		t.Fatal("expected expiry within 7 days to match")
	}
	if matchesVPNExpiringInDays("06/04/2026", 7, now) {
		t.Fatal("expected expiry beyond 7 days to not match")
	}
	if matchesVPNExpiringInDays("29/03/2026", 7, now) {
		t.Fatal("expected expired user to not match expiring filter")
	}
}

func TestFilterUserRowsByVPNState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 30, 10, 0, 0, 0, vpnFilterLocation())
	rows := []map[string]any{
		{
			"username":          "inactive-expiring",
			"last_vpn_login_at": "2026-02-20T10:00:00+07:00",
			"vpn_expire_at":     "05/04/2026",
		},
		{
			"username":          "active-expiring",
			"last_vpn_login_at": "2026-03-28T10:00:00+07:00",
			"vpn_expire_at":     "05/04/2026",
		},
		{
			"username":          "inactive-late",
			"last_vpn_login_at": "2026-02-20T10:00:00+07:00",
			"vpn_expire_at":     "20/04/2026",
		},
	}

	filtered := filterUserRowsByVPNState(rows, 30, 7, now)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(filtered))
	}
	if got := formatCell(filtered[0]["username"]); got != "inactive-expiring" {
		t.Fatalf("unexpected filtered user: %q", got)
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
