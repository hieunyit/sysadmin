package email

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"backend/internal/config"
)

func TestServiceRenderAccountCreatedTemplate(t *testing.T) {
	t.Parallel()

	svc, err := New(config.SMTPConfig{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "no-reply@example.com",
		FromName:    "VPN",
		TLSMode:     "starttls",
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	subject, textBody, htmlBody, err := svc.render("account_created", map[string]any{
		"RecipientName":     "Alice",
		"BrandName":         "Hệ thống SSO MBFS",
		"SupportContact":    "it-support@mobifonesolutions.vn",
		"LoginURL":          "https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account",
		"Username":          "alice",
		"Email":             "alice@example.com",
		"TemporaryPassword": "Mbfs@111",
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if subject == "" || textBody == "" || htmlBody == "" {
		t.Fatalf("expected rendered subject/text/html, got subject=%q text=%q html=%q", subject, textBody, htmlBody)
	}
	if !strings.Contains(htmlBody, "Đăng nhập và đổi mật khẩu") {
		t.Fatalf("expected login CTA in html body")
	}
	if !strings.Contains(textBody, "https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account") {
		t.Fatalf("expected login url in text body")
	}
	if strings.Contains(htmlBody, "Thông tin quyền truy cập VPN") {
		t.Fatalf("expected vpn section to be removed from account-created html body")
	}
	if strings.Contains(textBody, "Đã cấp quyền truy cập VPN") {
		t.Fatalf("expected vpn section to be removed from account-created text body")
	}
}

func TestServiceRenderAccountCreatedTemplateWithNotificationRecipient(t *testing.T) {
	t.Parallel()

	svc, err := New(config.SMTPConfig{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "no-reply@example.com",
		FromName:    "VPN",
		TLSMode:     "starttls",
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	subject, textBody, htmlBody, err := svc.render("account_created", map[string]any{
		"RecipientName":         "Admin nhận thông báo",
		"BrandName":             "Hệ thống SSO MBFS",
		"SupportContact":        "it-support@mobifonesolutions.vn",
		"LoginURL":              "https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account",
		"Username":              "ldap.user",
		"Email":                 "ldap.user@example.com",
		"NotificationRecipient": "notify@example.com",
		"RecipientIsAccount":    false,
		"TemporaryPassword":     "Mbfs@111",
		"VPNProvisioned":        false,
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if !strings.Contains(subject, "Thông báo tạo tài khoản cho ldap.user") {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if !strings.Contains(textBody, "Email nhận thông báo: notify@example.com") {
		t.Fatalf("expected notification recipient in text body")
	}
	if !strings.Contains(htmlBody, "Email nhận thông báo") {
		t.Fatalf("expected notification recipient row in html body")
	}
}

func TestServiceRenderLDAPAccountCreatedTemplate(t *testing.T) {
	t.Parallel()

	svc, err := New(config.SMTPConfig{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "no-reply@example.com",
		FromName:    "VPN",
		TLSMode:     "starttls",
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	subject, textBody, htmlBody, err := svc.render("account_created_ldap", map[string]any{
		"RecipientName":     "Bạn Linh",
		"BrandName":         "Hệ thống SSO MBFS",
		"SupportContact":    "it-support@mobifonesolutions.vn",
		"LoginURL":          "https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account",
		"WebmailURL":        "https://outlook.office365.com/",
		"Username":          "linhnn",
		"Email":             "linhnn@mobifonesolutions.vn",
		"EmployeeID":        "NV00123",
		"Department":        "Trung tâm Dịch vụ Chuyển đổi số",
		"OnboardDate":       "15/04/2026",
		"WorkAddress":       "Tòa nhà MobiFone, Hà Nội",
		"TemporaryPassword": "Mbfs@111",
		"VPNProvisioned":    true,
		"VPNGroup":          "vanhanh",
		"VPNExpireAt":       "27/07/2026",
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if got := subject; got != "[MobiFone Solutions] Thư chào mừng nhân sự mới - Bạn Linh" {
		t.Fatalf("unexpected subject: %q", got)
	}
	for _, expected := range []string{
		"https://outlook.office365.com/",
		"linhnn@mobifonesolutions.vn",
		"NV00123",
		"Trung tâm Dịch vụ Chuyển đổi số",
		"15/04/2026",
		"Tòa nhà MobiFone, Hà Nội",
		"https://zalo.me/g/yjndoy063",
		"hr@mobifonesolutions.vn",
		"0932325002",
		"Công ty Cổ phần Giải pháp Số MobiFone(MobiFone Solutions)",
	} {
		if !strings.Contains(textBody, expected) {
			t.Fatalf("expected %q in ldap text body", expected)
		}
	}
	if !strings.Contains(htmlBody, "Chào mừng bạn gia nhập") {
		t.Fatalf("expected onboarding heading in ldap html body")
	}
	if !strings.Contains(htmlBody, "https://mobifonesolutions.vn/images/mbfs-logo.svg") {
		t.Fatalf("expected official logo url in ldap html body")
	}
	if strings.Contains(htmlBody, "Thông tin quyền truy cập VPN") {
		t.Fatalf("expected vpn section to be omitted from ldap html body")
	}
	if strings.Contains(htmlBody, "Nếu cần hỗ trợ thêm") {
		t.Fatalf("expected support box to be removed from ldap html body")
	}
	for _, unexpected := range []string{"&#128205;", "&#128222;", "&#127760;", "&#128100;", "&#128231;"} {
		if strings.Contains(htmlBody, unexpected) {
			t.Fatalf("expected icons to be removed from ldap html body")
		}
	}
	for _, expected := range []string{
		"Hướng dẫn chấm công",
		"https://quickchart.io/qr?text=https%3A%2F%2Fzalo.me%2Fg%2Fyjndoy063&size=180",
		"Quét mã QR để tham gia nhóm Zalo",
		"Trân trọng,",
		"Công ty Cổ phần Giải pháp Số MobiFone",
		"Số 38 Phố Phan Đình Phùng",
		"hr@mobifonesolutions.vn",
		"0932325002",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("expected %q in ldap html body", expected)
		}
	}
}

func TestServiceRenderVPNProvisionedTemplate(t *testing.T) {
	t.Parallel()

	svc, err := New(config.SMTPConfig{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "no-reply@example.com",
		FromName:    "VPN",
		TLSMode:     "starttls",
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	subject, textBody, htmlBody, err := svc.render("vpn_provisioned", map[string]any{
		"RecipientName":     "Alice",
		"BrandName":         "Hệ thống SSO MBFS",
		"SupportContact":    "it-support@mobifonesolutions.vn",
		"LoginURL":          "https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account",
		"Username":          "alice",
		"Email":             "alice@example.com",
		"VPNGroup":          "partners-vpn",
		"VPNExpireAt":       "20/04/2026",
		"RecipientIsAccount": true,
	})
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	for _, expected := range []string{
		"Thông tin quyền truy cập VPN",
		"partners-vpn",
		"20/04/2026",
		"Sử dụng tài khoản SSO MBFS",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("expected %q in html body", expected)
		}
		if !strings.Contains(textBody, expected) && expected != "Thông tin quyền truy cập VPN" {
			t.Fatalf("expected %q in text body", expected)
		}
	}
	if !strings.Contains(subject, "Thông tin quyền truy cập VPN") {
		t.Fatalf("unexpected subject: %q", subject)
	}
}

func TestBuildMessageIncludesDeliverabilityHeaders(t *testing.T) {
	t.Parallel()

	svc, err := New(config.SMTPConfig{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		FromAddress: "no-reply@example.com",
		FromName:    "VPN",
		TLSMode:     "starttls",
	}, zerolog.Nop())
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	msg, err := svc.buildMessage([]string{"alice@example.com"}, "Subject", "text body", "<p>html body</p>")
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	raw := string(msg)
	for _, header := range []string{"Date:", "Message-ID:", "Content-Language: vi", "X-Mailer: backend", "Auto-Submitted: auto-generated"} {
		if !strings.Contains(raw, header) {
			t.Fatalf("expected header %q in message", header)
		}
	}
}
