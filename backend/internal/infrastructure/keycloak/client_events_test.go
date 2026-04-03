package keycloak

import "testing"

func TestSelectLatestVPNEventTimePrefersExactClientID(t *testing.T) {
	t.Parallel()

	events := []keycloakEventPayload{
		{
			Time:     1770180633542,
			ClientID: "account-console",
		},
		{
			Time:     1770280633542,
			ClientID: "https://vpn.mbfs.vn/saml/metadata",
		},
		{
			Time:     1770380633542,
			ClientID: "other-client",
		},
	}

	got := selectLatestVPNEventTime(events, "https://vpn.mbfs.vn/saml/metadata")
	if got != "2026-02-05T15:37:13+07:00" {
		t.Fatalf("unexpected latest vpn event time: %q", got)
	}
}

func TestSelectLatestVPNEventTimeFallsBackToRedirectURIHost(t *testing.T) {
	t.Parallel()

	events := []keycloakEventPayload{
		{
			Time:     1770180633542,
			ClientID: "broker",
			Details: map[string]string{
				"redirect_uri": "https://vpn.mbfs.vn/saml/acs",
			},
		},
	}

	got := selectLatestVPNEventTime(events, "https://vpn.mbfs.vn/saml/metadata")
	if got != "2026-02-04T11:50:33+07:00" {
		t.Fatalf("unexpected vpn event time from redirect_uri fallback: %q", got)
	}
}

func TestSelectLatestVPNEventTimeReturnsEmptyWhenNoMatch(t *testing.T) {
	t.Parallel()

	events := []keycloakEventPayload{
		{
			Time:     1770180633542,
			ClientID: "account-console",
		},
	}

	if got := selectLatestVPNEventTime(events, "https://vpn.mbfs.vn/saml/metadata"); got != "" {
		t.Fatalf("expected empty last vpn login, got %q", got)
	}
}
