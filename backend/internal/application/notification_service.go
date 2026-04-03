package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"backend/internal/domain/services"
)

type NotificationService struct {
	mailer         services.MailTemplateSender
	ldapMailer     services.MailTemplateSender
	logger         zerolog.Logger
	brandName      string
	supportContact string
	loginURL       string
}

type accessNotificationEntry struct {
	Summary string
}

type accountCreatedTemplateData struct {
	RecipientName         string
	BrandName             string
	SupportContact        string
	LoginURL              string
	WebmailURL            string
	Username              string
	Email                 string
	EmployeeID            string
	Department            string
	OnboardDate           string
	WorkAddress           string
	NotificationRecipient string
	RecipientIsAccount    bool
	TemporaryPassword     string
}

type vpnProvisionedTemplateData struct {
	RecipientName         string
	BrandName             string
	SupportContact        string
	LoginURL              string
	Username              string
	Email                 string
	NotificationRecipient string
	RecipientIsAccount    bool
	VPNGroup              string
	VPNExpireAt           string
}

type userStatusTemplateData struct {
	RecipientName  string
	BrandName      string
	SupportContact string
	Username       string
	Email          string
	StatusLabel    string
	Actor          string
	ChangedAt      string
}

type accessChangedTemplateData struct {
	RecipientName  string
	BrandName      string
	SupportContact string
	SubjectLabel   string
	ActionLabel    string
	Entries        []accessNotificationEntry
	GroupContext   string
	Actor          string
	ChangedAt      string
}

func NewNotificationService(mailer services.MailTemplateSender, ldapMailer services.MailTemplateSender, logger zerolog.Logger, brandName, supportContact, loginURL string) *NotificationService {
	return &NotificationService{
		mailer:         mailer,
		ldapMailer:     ldapMailer,
		logger:         logger.With().Str("component", "notifications").Logger(),
		brandName:      strings.TrimSpace(brandName),
		supportContact: strings.TrimSpace(supportContact),
		loginURL:       strings.TrimSpace(loginURL),
	}
}

func (s *NotificationService) Enabled() bool {
	return s != nil && ((s.mailer != nil && s.mailer.Enabled()) || (s.ldapMailer != nil && s.ldapMailer.Enabled()))
}

func (s *NotificationService) TrySendAccountCreated(ctx context.Context, user services.KeycloakUser, temporaryPassword string, notificationEmail string) []string {
	mailer := s.accountCreatedMailer(user.IdentitySource)
	if mailer == nil || !mailer.Enabled() {
		return nil
	}
	recipient := strings.TrimSpace(notificationEmail)
	if recipient == "" {
		recipient = strings.TrimSpace(user.Email)
	}
	if recipient == "" {
		return nil
	}

	data := accountCreatedTemplateData{
		RecipientName:         notificationRecipientName(user),
		BrandName:             s.effectiveBrandName(),
		SupportContact:        s.effectiveSupportContact(),
		LoginURL:              s.loginURL,
		WebmailURL:            notificationWebmailURL(),
		Username:              strings.TrimSpace(user.Username),
		Email:                 strings.TrimSpace(user.Email),
		EmployeeID:            notificationEmployeeID(user),
		Department:            notificationDepartment(user),
		OnboardDate:           notificationOnboardDate(user),
		WorkAddress:           notificationWorkAddress(user),
		NotificationRecipient: recipient,
		RecipientIsAccount:    strings.EqualFold(recipient, strings.TrimSpace(user.Email)),
		TemporaryPassword:     strings.TrimSpace(temporaryPassword),
	}
	templateName := "account_created"
	if strings.EqualFold(strings.TrimSpace(user.IdentitySource), "ldap") {
		templateName = "account_created_ldap"
	}
	if err := s.trySendWithMailer(ctx, mailer, templateName, []string{recipient}, data, map[string]string{
		"username": data.Username,
	}); err != nil {
		return []string{fmt.Sprintf("Tài khoản đã được tạo nhưng không gửi được email thông tin tài khoản tới %s: %v", recipient, err)}
	}
	return nil
}

func (s *NotificationService) TrySendVPNProvisioned(ctx context.Context, user services.KeycloakUser, vpnGroup string) []string {
	mailer := s.accountCreatedMailer(user.IdentitySource)
	if mailer == nil || !mailer.Enabled() {
		return nil
	}
	recipient := strings.TrimSpace(user.Email)
	if recipient == "" {
		return nil
	}

	data := vpnProvisionedTemplateData{
		RecipientName:         notificationRecipientName(user),
		BrandName:             s.effectiveBrandName(),
		SupportContact:        s.effectiveSupportContact(),
		LoginURL:              s.loginURL,
		Username:              strings.TrimSpace(user.Username),
		Email:                 strings.TrimSpace(user.Email),
		NotificationRecipient: recipient,
		RecipientIsAccount:    strings.EqualFold(recipient, strings.TrimSpace(user.Email)),
		VPNGroup:              strings.TrimSpace(vpnGroup),
		VPNExpireAt:           notificationVPNExpireAt(user),
	}
	if err := s.trySendWithMailer(ctx, mailer, "vpn_provisioned", []string{recipient}, data, map[string]string{
		"username": data.Username,
		"group":    data.VPNGroup,
	}); err != nil {
		return []string{fmt.Sprintf("Quyền truy cập VPN đã được cấp nhưng không gửi được email thông tin VPN tới %s: %v", recipient, err)}
	}
	return nil
}

func (s *NotificationService) TrySendUserStatusChanged(ctx context.Context, user services.KeycloakUser, actor string) {
	if !s.Enabled() {
		return
	}
	recipient := strings.TrimSpace(user.Email)
	if recipient == "" {
		return
	}

	data := userStatusTemplateData{
		RecipientName:  notificationRecipientName(user),
		BrandName:      s.effectiveBrandName(),
		SupportContact: s.effectiveSupportContact(),
		Username:       strings.TrimSpace(user.Username),
		Email:          recipient,
		StatusLabel:    boolLabel(user.Enabled, "đang hoạt động", "đã vô hiệu hóa"),
		Actor:          notificationActor(actor),
		ChangedAt:      notificationTimestamp(),
	}
	_ = s.trySend(ctx, "user_status_changed", []string{recipient}, data, map[string]string{
		"username": data.Username,
		"status":   data.StatusLabel,
	})
}

func (s *NotificationService) TrySendVPNAccessChanged(ctx context.Context, recipients []services.KeycloakUser, subjectType, subject, mode, actor string, entries []accessNotificationEntry) {
	if !s.Enabled() || len(entries) == 0 {
		return
	}

	recipients = uniqueUsersByEmail(recipients)
	if len(recipients) == 0 {
		return
	}

	subjectType = strings.TrimSpace(subjectType)
	subject = strings.TrimSpace(subject)
	groupContext := ""
	subjectLabel := "người dùng " + subject
	if subjectType == "group" {
		subjectLabel = "nhóm " + subject
		groupContext = subject
	}
	actionLabel := "cập nhật"
	if strings.EqualFold(strings.TrimSpace(mode), "remove") {
		actionLabel = "gỡ bỏ"
	}

	for _, recipient := range recipients {
		email := strings.TrimSpace(recipient.Email)
		if email == "" {
			continue
		}
		data := accessChangedTemplateData{
			RecipientName:  notificationRecipientName(recipient),
			BrandName:      s.effectiveBrandName(),
			SupportContact: s.effectiveSupportContact(),
			SubjectLabel:   subjectLabel,
			ActionLabel:    actionLabel,
			Entries:        entries,
			GroupContext:   groupContext,
			Actor:          notificationActor(actor),
			ChangedAt:      notificationTimestamp(),
		}
		_ = s.trySend(ctx, "vpn_access_changed", []string{email}, data, map[string]string{
			"username":     strings.TrimSpace(recipient.Username),
			"subject_type": subjectType,
			"subject":      subject,
			"mode":         mode,
		})
	}
}

func (s *NotificationService) trySend(ctx context.Context, templateName string, to []string, data any, fields map[string]string) error {
	return s.trySendWithMailer(ctx, s.mailer, templateName, to, data, fields)
}

func (s *NotificationService) trySendWithMailer(ctx context.Context, mailer services.MailTemplateSender, templateName string, to []string, data any, fields map[string]string) error {
	if !s.Enabled() {
		return nil
	}
	if mailer == nil || !mailer.Enabled() {
		return nil
	}
	if err := mailer.SendTemplate(ctx, templateName, to, data); err != nil {
		event := s.logger.Error().Err(err).Str("template", templateName)
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			event = event.Str(key, fields[key])
		}
		event.Msg("send notification email failed")
		return err
	}
	return nil
}

func (s *NotificationService) accountCreatedMailer(identitySource string) services.MailTemplateSender {
	if strings.EqualFold(strings.TrimSpace(identitySource), "ldap") && s.ldapMailer != nil && s.ldapMailer.Enabled() {
		return s.ldapMailer
	}
	return s.mailer
}

func notificationRecipientName(user services.KeycloakUser) string {
	if name := strings.TrimSpace(strings.TrimSpace(user.LastName) + " " + strings.TrimSpace(user.FirstName)); name != "" {
		return name
	}
	if user.Attributes != nil {
		if name := strings.TrimSpace(user.Attributes["fullName"]); name != "" {
			return name
		}
	}
	if name := strings.TrimSpace(user.DisplayName); name != "" {
		return name
	}
	if username := strings.TrimSpace(user.Username); username != "" {
		return username
	}
	return "user"
}

func notificationActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}

func notificationTimestamp() string {
	loc := time.FixedZone("UTC+7", 7*60*60)
	return time.Now().In(loc).Format("02/01/2006 15:04:05 +07")
}

func (s *NotificationService) effectiveBrandName() string {
	if strings.TrimSpace(s.brandName) != "" {
		return strings.TrimSpace(s.brandName)
	}
	return "Hệ thống SSO MBFS"
}

func (s *NotificationService) effectiveSupportContact() string {
	if strings.TrimSpace(s.supportContact) != "" {
		return strings.TrimSpace(s.supportContact)
	}
	return "it-support@mobifonesolutions.vn"
}

func notificationVPNExpireAt(_ services.KeycloakUser) string {
	return ""
}

func notificationEmployeeID(user services.KeycloakUser) string {
	if user.Attributes == nil {
		return ""
	}
	return strings.TrimSpace(user.Attributes["employeeID"])
}

func notificationDepartment(user services.KeycloakUser) string {
	if user.Attributes == nil {
		return ""
	}
	return strings.TrimSpace(user.Attributes["department"])
}

func notificationOnboardDate(user services.KeycloakUser) string {
	if user.Attributes == nil {
		return ""
	}
	return strings.TrimSpace(user.Attributes["onboardDate"])
}

func notificationWorkAddress(user services.KeycloakUser) string {
	if user.Attributes == nil {
		return ""
	}
	return strings.TrimSpace(user.Attributes["workAddress"])
}

func notificationWebmailURL() string {
	return "https://outlook.office365.com/"
}

func boolLabel(v bool, trueLabel, falseLabel string) string {
	if v {
		return trueLabel
	}
	return falseLabel
}

func uniqueUsersByEmail(in []services.KeycloakUser) []services.KeycloakUser {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]services.KeycloakUser, 0, len(in))
	for _, user := range in {
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, user)
	}
	return out
}
