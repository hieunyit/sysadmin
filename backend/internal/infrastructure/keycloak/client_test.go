package keycloak

import "testing"

func TestAttributesToKeycloakSkipsEmptyValues(t *testing.T) {
	t.Parallel()

	got := attributesToKeycloak(map[string]string{
		"fullName":    "Alice User",
		"companyName": "",
		"State":       "   ",
		" employeeID": " 12345 ",
		"":            "ignored",
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 attributes, got %#v", got)
	}
	if _, ok := got["companyName"]; ok {
		t.Fatalf("expected empty companyName to be omitted")
	}
	if _, ok := got["State"]; ok {
		t.Fatalf("expected blank State to be omitted")
	}
	if got["fullName"][0] != "Alice User" {
		t.Fatalf("unexpected fullName: %#v", got["fullName"])
	}
	if got["employeeID"][0] != "12345" {
		t.Fatalf("unexpected employeeID: %#v", got["employeeID"])
	}
}
