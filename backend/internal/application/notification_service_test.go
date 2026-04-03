package application

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"backend/internal/domain/services"
)

type fakeMailTemplateSender struct {
	enabled bool
	calls   []fakeMailCall
	sendFn  func(context.Context, string, []string, any) error
}

type fakeMailCall struct {
	Template string
	To       []string
	Data     any
}

func (f *fakeMailTemplateSender) Enabled() bool {
	return f != nil && f.enabled
}

func (f *fakeMailTemplateSender) SendTemplate(ctx context.Context, templateName string, to []string, data any) error {
	f.calls = append(f.calls, fakeMailCall{
		Template: templateName,
		To:       append([]string(nil), to...),
		Data:     data,
	})
	if f.sendFn != nil {
		return f.sendFn(ctx, templateName, to, data)
	}
	return nil
}

func TestNotificationServiceTrySendAccountCreated(t *testing.T) {
	t.Parallel()

	mailer := &fakeMailTemplateSender{enabled: true}
	svc := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")
	warnings := svc.TrySendAccountCreated(context.Background(), services.KeycloakUser{
		Username:       "alice",
		Email:          "alice@example.com",
		DisplayName:    "Alice",
		IdentitySource: "local",
		Enabled:        true,
	}, "Mbfs@111", "")
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	if len(mailer.calls) != 1 {
		t.Fatalf("expected 1 mail call, got %d", len(mailer.calls))
	}
	if got := mailer.calls[0].Template; got != "account_created" {
		t.Fatalf("unexpected template: %q", got)
	}
	if got := mailer.calls[0].To[0]; got != "alice@example.com" {
		t.Fatalf("unexpected recipient: %q", got)
	}
	data, ok := mailer.calls[0].Data.(accountCreatedTemplateData)
	if !ok {
		t.Fatalf("unexpected template data type: %T", mailer.calls[0].Data)
	}
	if got := data.LoginURL; got != "https://sso.mbfs.vn" {
		t.Fatalf("unexpected login url: %q", got)
	}
	if !data.RecipientIsAccount {
		t.Fatalf("expected recipient to be account owner")
	}
}

func TestNotificationServiceTrySendAccountCreatedUsesNotificationEmailOverride(t *testing.T) {
	t.Parallel()

	mailer := &fakeMailTemplateSender{enabled: true}
	svc := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")
	warnings := svc.TrySendAccountCreated(context.Background(), services.KeycloakUser{
		Username:    "alice",
		Email:       "alice@account.example.com",
		DisplayName: "Alice",
	}, "Mbfs@111", "notify@example.com")
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	if len(mailer.calls) != 1 {
		t.Fatalf("expected 1 mail call, got %d", len(mailer.calls))
	}
	if got := mailer.calls[0].To[0]; got != "notify@example.com" {
		t.Fatalf("unexpected recipient: %q", got)
	}
	data, ok := mailer.calls[0].Data.(accountCreatedTemplateData)
	if !ok {
		t.Fatalf("unexpected template data type: %T", mailer.calls[0].Data)
	}
	if got := data.Email; got != "alice@account.example.com" {
		t.Fatalf("unexpected account email in template data: %q", got)
	}
	if data.RecipientIsAccount {
		t.Fatalf("expected recipient to differ from account email")
	}
	if got := data.NotificationRecipient; got != "notify@example.com" {
		t.Fatalf("unexpected notification recipient in template data: %q", got)
	}
}

func TestNotificationServiceTrySendAccountCreatedUsesLDAPPMailer(t *testing.T) {
	t.Parallel()

	primaryMailer := &fakeMailTemplateSender{enabled: true}
	ldapMailer := &fakeMailTemplateSender{enabled: true}
	svc := NewNotificationService(primaryMailer, ldapMailer, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")
	warnings := svc.TrySendAccountCreated(context.Background(), services.KeycloakUser{
		Username:       "ldap.user",
		Email:          "ldap.user@example.com",
		DisplayName:    "LDAP User",
		IdentitySource: "ldap",
		Attributes: map[string]string{
			"department":  "Trung tâm Dịch vụ Chuyển đổi số",
			"onboardDate": "15/04/2026",
			"workAddress": "Tòa nhà MobiFone, Hà Nội",
		},
	}, "Mbfs@111", "notify@example.com")
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	if len(primaryMailer.calls) != 0 {
		t.Fatalf("expected primary mailer to be unused, got %d calls", len(primaryMailer.calls))
	}
	if len(ldapMailer.calls) != 1 {
		t.Fatalf("expected ldap mailer to be used once, got %d calls", len(ldapMailer.calls))
	}
	if got := ldapMailer.calls[0].Template; got != "account_created_ldap" {
		t.Fatalf("unexpected ldap template: %q", got)
	}
	data, ok := ldapMailer.calls[0].Data.(accountCreatedTemplateData)
	if !ok {
		t.Fatalf("unexpected template data type: %T", ldapMailer.calls[0].Data)
	}
	if got := data.Department; got != "Trung tâm Dịch vụ Chuyển đổi số" {
		t.Fatalf("unexpected department: %q", got)
	}
	if got := data.OnboardDate; got != "15/04/2026" {
		t.Fatalf("unexpected onboard date: %q", got)
	}
	if got := data.WorkAddress; got != "Tòa nhà MobiFone, Hà Nội" {
		t.Fatalf("unexpected work address: %q", got)
	}
}

func TestNotificationServiceTrySendVPNProvisioned(t *testing.T) {
	t.Parallel()

	mailer := &fakeMailTemplateSender{enabled: true}
	svc := NewNotificationService(mailer, nil, zerolog.Nop(), "Hệ thống SSO MBFS", "it.support@mbfs.vn", "https://sso.mbfs.vn")
	warnings := svc.TrySendVPNProvisioned(context.Background(), services.KeycloakUser{
		Username: "alice",
		Email:    "alice@example.com",
	}, "partners-vpn")
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if len(mailer.calls) != 1 {
		t.Fatalf("expected 1 mail call, got %d", len(mailer.calls))
	}
	if got := mailer.calls[0].Template; got != "vpn_provisioned" {
		t.Fatalf("unexpected template: %q", got)
	}
	if got := mailer.calls[0].To[0]; got != "alice@example.com" {
		t.Fatalf("unexpected recipient: %q", got)
	}
	data, ok := mailer.calls[0].Data.(vpnProvisionedTemplateData)
	if !ok {
		t.Fatalf("unexpected template data type: %T", mailer.calls[0].Data)
	}
	if !data.RecipientIsAccount {
		t.Fatalf("expected vpn mail recipient to be account owner")
	}
}

func TestNotificationRecipientNamePrefersLastNameThenFirstName(t *testing.T) {
	t.Parallel()

	got := notificationRecipientName(services.KeycloakUser{
		Username:    "linhnn",
		FirstName:   "Linh",
		LastName:    "Nguyễn Ngọc",
		DisplayName: "Linh Nguyễn Ngọc",
		Attributes: map[string]string{
			"fullName": "Nguyễn Ngọc Linh",
		},
	})
	if got != "Nguyễn Ngọc Linh" {
		t.Fatalf("unexpected recipient name: %q", got)
	}
}
