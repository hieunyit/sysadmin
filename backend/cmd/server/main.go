package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"backend/internal/application"
	"backend/internal/bootstrap"
	"backend/internal/config"
	"backend/internal/infrastructure/email"
	"backend/internal/infrastructure/keycloak"
	"backend/internal/infrastructure/openvpn"
	"backend/internal/infrastructure/postgres"
	transporthttp "backend/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		fmt.Fprintln(os.Stderr, "hint: load env before run, e.g. `set -a; source .env; set +a`")
		os.Exit(1)
	}

	logger := bootstrap.NewLogger(cfg.LogLevel)
	validate := validator.New()

	keycloakClient := keycloak.New(cfg.Keycloak)
	openvpnClient := openvpn.New(cfg.OpenVPN)
	adminPortalStore, err := postgres.NewAdminPortalStore(context.Background(), cfg.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("init admin portal store failed")
	}
	if adminPortalStore != nil {
		defer adminPortalStore.Close()
	}
	var adminPortalService *application.AdminPortalService
	if adminPortalStore != nil {
		adminPortalService = application.NewAdminPortalService(adminPortalStore)
	}
	logSMTPProfile(logger, "primary", cfg.SMTP)
	logSMTPProfile(logger, "ldap", cfg.SMTPLDAP)
	emailService, err := email.New(cfg.SMTP, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("init email service failed")
	}
	ldapEmailService, err := email.New(cfg.SMTPLDAP, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("init ldap email service failed")
	}
	notifications := application.NewNotificationService(emailService, ldapEmailService, logger, cfg.SMTP.BrandName, cfg.SMTP.SupportContact, cfg.SMTP.LoginURL)

	services := &application.Services{
		User:               application.NewUserService(validate, keycloakClient, notifications),
		Group:              application.NewGroupService(validate, keycloakClient),
		OpenVPNAdmin:       application.NewOpenVPNAdminService(openvpnClient, notifications),
		OpenVPNProvisioner: application.NewOpenVPNProvisioningService(openvpnClient, keycloakClient, notifications),
		AdminPortal:        adminPortalService,
	}

	apiServer := transporthttp.NewServer(cfg, logger, services)

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info().Str("addr", cfg.HTTP.Addr).Msg("http server starting")
		if err := apiServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("http server failed")
		}
	}()

	<-runCtx.Done()
	logger.Info().Msg("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("http shutdown failed")
	}

	logger.Info().Msg("shutdown complete")
}

func logSMTPProfile(logger zerolog.Logger, profile string, cfg config.SMTPConfig) {
	event := logger.Info().
		Str("profile", profile).
		Bool("enabled", cfg.Enabled).
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("tls_mode", cfg.TLSMode).
		Str("from_address", cfg.FromAddress).
		Str("from_name", cfg.FromName).
		Str("username", maskSMTPIdentity(cfg.Username))
	if cfg.Enabled {
		event.Msg("smtp profile loaded")
		return
	}
	event.Msg("smtp profile disabled")
}

func maskSMTPIdentity(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if at := strings.Index(value, "@"); at > 1 {
		return value[:2] + "***" + value[at:]
	}
	if len(value) <= 2 {
		return "**"
	}
	return value[:2] + "***"
}
