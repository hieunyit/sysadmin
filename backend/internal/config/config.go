package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv   string
	LogLevel string
	HTTP     HTTPConfig
	Keycloak KeycloakConfig
	OpenVPN  OpenVPNConfig
	SMTP     SMTPConfig
	SMTPLDAP SMTPConfig
	Authz    AuthzConfig
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type KeycloakConfig struct {
	BaseURL           string
	Realm             string
	ClientID          string
	ClientSecret      string
	TokenURL          string
	VPNEventClientID  string
	LookupConcurrency int
	Timeout           time.Duration
	LDAPComponentID   string
	ComponentLockFile string
}

type OpenVPNConfig struct {
	BaseURL          string
	Username         string
	Password         string
	Timeout          time.Duration
	InsecureSkipTLS  bool
	UseShadowObjects bool
}

type SMTPConfig struct {
	Enabled            bool
	Host               string
	Port               int
	Username           string
	Password           string
	FromAddress        string
	FromName           string
	BrandName          string
	SupportContact     string
	LoginURL           string
	TLSMode            string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

type AuthzConfig struct {
	AdminToken string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:   getenv("APP_ENV", "dev"),
		LogLevel: getenv("LOG_LEVEL", "info"),
	}

	required := []string{
		"KEYCLOAK_BASE_URL",
		"KEYCLOAK_REALM",
		"KEYCLOAK_CLIENT_ID",
		"KEYCLOAK_CLIENT_SECRET",
		"OPENVPN_BASE_URL",
		"OPENVPN_USERNAME",
		"OPENVPN_PASSWORD",
		"ADMIN_API_TOKEN",
	}
	for _, k := range required {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			return Config{}, fmt.Errorf("missing required env: %s", k)
		}
	}

	readTimeout, err := getDuration("HTTP_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := getDuration("HTTP_WRITE_TIMEOUT", 20*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := getDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.HTTP = HTTPConfig{
		Addr:            getenv("HTTP_ADDR", ":8080"),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
	}

	keycloakBaseURL := os.Getenv("KEYCLOAK_BASE_URL")
	keycloakRealm := os.Getenv("KEYCLOAK_REALM")
	keycloakTimeout, err := getDuration("KEYCLOAK_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	keycloakLookupConcurrency, err := getInt("KEYCLOAK_LOOKUP_CONCURRENCY", 12)
	if err != nil {
		return Config{}, err
	}
	if keycloakLookupConcurrency <= 0 {
		return Config{}, fmt.Errorf("invalid int for KEYCLOAK_LOOKUP_CONCURRENCY: must be greater than 0")
	}
	cfg.Keycloak = KeycloakConfig{
		BaseURL:      keycloakBaseURL,
		Realm:        keycloakRealm,
		ClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		TokenURL: getenv(
			"KEYCLOAK_TOKEN_URL",
			defaultKeycloakTokenURL(keycloakBaseURL, keycloakRealm),
		),
		VPNEventClientID:  getenv("KEYCLOAK_VPN_EVENT_CLIENT_ID", "https://vpn.mbfs.vn/saml/metadata"),
		LookupConcurrency: keycloakLookupConcurrency,
		Timeout:           keycloakTimeout,
		LDAPComponentID:   getenv("KEYCLOAK_LDAP_COMPONENT_ID", ""),
		ComponentLockFile: getenv("KEYCLOAK_COMPONENT_LOCK_FILE", "/tmp/backend-keycloak-component.lock"),
	}

	openvpnTimeout, err := getDuration("OPENVPN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	openvpnInsecureSkipTLS, err := getBool("OPENVPN_INSECURE_SKIP_TLS", false)
	if err != nil {
		return Config{}, err
	}
	openvpnUseShadowObjects, err := getBool("OPENVPN_USE_SHADOW_OBJECTS", true)
	if err != nil {
		return Config{}, err
	}
	cfg.OpenVPN = OpenVPNConfig{
		BaseURL:          os.Getenv("OPENVPN_BASE_URL"),
		Username:         os.Getenv("OPENVPN_USERNAME"),
		Password:         os.Getenv("OPENVPN_PASSWORD"),
		Timeout:          openvpnTimeout,
		InsecureSkipTLS:  openvpnInsecureSkipTLS,
		UseShadowObjects: openvpnUseShadowObjects,
	}

	cfg.SMTP, err = loadSMTPConfig(
		"SMTP",
		SMTPConfig{
			FromName:       "Hệ thống SSO MBFS",
			BrandName:      "Hệ thống SSO MBFS",
			SupportContact: "it-support@mobifonesolutions.vn",
			LoginURL:       "https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account",
			TLSMode:        "starttls",
			Port:           587,
			Timeout:        10 * time.Second,
		},
	)
	if err != nil {
		return Config{}, err
	}
	cfg.SMTPLDAP, err = loadSMTPConfig(
		"SMTP_LDAP",
		cfg.SMTP,
	)
	if err != nil {
		return Config{}, err
	}

	cfg.Authz = AuthzConfig{
		AdminToken: os.Getenv("ADMIN_API_TOKEN"),
	}

	return cfg, nil
}

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func getBool(k string, def bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def, nil
	}
	if strings.EqualFold(v, "true") {
		return true, nil
	}
	if strings.EqualFold(v, "false") {
		return false, nil
	}
	return false, fmt.Errorf("invalid bool for %s: %s", k, v)
}

func getDuration(k string, def time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %v", k, err)
	}
	return d, nil
}

func getInt(k string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def, nil
	}
	out, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid int for %s: %s", k, v)
	}
	return out, nil
}

func loadSMTPConfig(prefix string, defaults SMTPConfig) (SMTPConfig, error) {
	enabled, err := getBool(prefix+"_ENABLED", false)
	if err != nil {
		return SMTPConfig{}, err
	}
	defaultTimeout := defaults.Timeout
	if defaultTimeout <= 0 {
		defaultTimeout = 10 * time.Second
	}
	timeout, err := getDuration(prefix+"_TIMEOUT", defaultTimeout)
	if err != nil {
		return SMTPConfig{}, err
	}
	defaultPort := defaults.Port
	if defaultPort <= 0 {
		defaultPort = 587
	}
	port, err := getInt(prefix+"_PORT", defaultPort)
	if err != nil {
		return SMTPConfig{}, err
	}
	if enabled && port <= 0 {
		return SMTPConfig{}, fmt.Errorf("invalid int for %s_PORT: must be greater than 0", prefix)
	}
	defaultTLSMode := strings.TrimSpace(defaults.TLSMode)
	if defaultTLSMode == "" {
		defaultTLSMode = "starttls"
	}
	tlsMode := strings.ToLower(strings.TrimSpace(getenv(prefix+"_TLS_MODE", defaultTLSMode)))
	switch tlsMode {
	case "starttls", "direct", "none":
	default:
		return SMTPConfig{}, fmt.Errorf("invalid %s_TLS_MODE: must be one of starttls, direct, none", prefix)
	}
	insecureSkipVerify, err := getBool(prefix+"_INSECURE_SKIP_VERIFY", defaults.InsecureSkipVerify)
	if err != nil {
		return SMTPConfig{}, err
	}

	cfg := SMTPConfig{
		Enabled:            enabled,
		Host:               strings.TrimSpace(getenv(prefix+"_HOST", defaults.Host)),
		Port:               port,
		Username:           strings.TrimSpace(getenv(prefix+"_USERNAME", defaults.Username)),
		Password:           getenv(prefix+"_PASSWORD", defaults.Password),
		FromAddress:        strings.TrimSpace(getenv(prefix+"_FROM_ADDRESS", defaults.FromAddress)),
		FromName:           strings.TrimSpace(getenv(prefix+"_FROM_NAME", defaults.FromName)),
		BrandName:          strings.TrimSpace(getenv(prefix+"_BRAND_NAME", defaults.BrandName)),
		SupportContact:     strings.TrimSpace(getenv(prefix+"_SUPPORT_CONTACT", defaults.SupportContact)),
		LoginURL:           strings.TrimSpace(getenv(prefix+"_LOGIN_URL", defaults.LoginURL)),
		TLSMode:            tlsMode,
		InsecureSkipVerify: insecureSkipVerify,
		Timeout:            timeout,
	}
	if cfg.Enabled {
		switch {
		case cfg.Host == "":
			return SMTPConfig{}, fmt.Errorf("missing required env: %s_HOST", prefix)
		case cfg.FromAddress == "":
			return SMTPConfig{}, fmt.Errorf("missing required env: %s_FROM_ADDRESS", prefix)
		}
	}
	return cfg, nil
}

func defaultKeycloakTokenURL(baseURL, realm string) string {
	raw := strings.TrimSpace(baseURL)
	if raw == "" || strings.TrimSpace(realm) == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(raw, "/") + "/realms/" + realm + "/protocol/openid-connect/token"
	}
	p := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(p, "/admin") {
		p = strings.TrimSuffix(p, "/admin")
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		u.Path = path.Join("/", "realms", realm, "protocol", "openid-connect", "token")
	} else {
		u.Path = path.Join("/", p, "realms", realm, "protocol", "openid-connect", "token")
	}
	return u.String()
}
