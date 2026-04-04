package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"backend/internal/application"
	"backend/internal/config"
	domainerr "backend/internal/domain/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminPortalStore struct {
	pool *pgxpool.Pool
}

func NewAdminPortalStore(ctx context.Context, cfg config.DatabaseConfig) (*AdminPortalStore, error) {
	if !cfg.Enabled || strings.TrimSpace(cfg.URL) == "" {
		return nil, nil
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxOpenConns
	poolConfig.MinConns = cfg.MinOpenConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	store := &AdminPortalStore{pool: pool}
	if err := store.init(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *AdminPortalStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *AdminPortalStore) init(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return nil
	}
	for _, stmt := range schemaStatements() {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("apply admin portal schema: %w", err)
		}
	}
	return s.seed(ctx)
}

func schemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS admin_keycloak_users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			email_verified BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			groups JSONB NOT NULL DEFAULT '[]'::jsonb,
			roles JSONB NOT NULL DEFAULT '[]'::jsonb,
			attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
			required_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
			password_hash TEXT NOT NULL DEFAULT '',
			password_temporary BOOLEAN NOT NULL DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS admin_keycloak_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			parent_id TEXT REFERENCES admin_keycloak_groups(id) ON DELETE CASCADE,
			roles JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS admin_keycloak_roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			composite BOOLEAN NOT NULL DEFAULT FALSE,
			client_role BOOLEAN NOT NULL DEFAULT FALSE,
			container_id TEXT NOT NULL DEFAULT 'master',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS admin_keycloak_sessions (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			user_id TEXT NOT NULL,
			ip_address TEXT NOT NULL,
			start_at TIMESTAMPTZ NOT NULL,
			last_access_at TIMESTAMPTZ NOT NULL,
			clients JSONB NOT NULL DEFAULT '{}'::jsonb,
			active BOOLEAN NOT NULL DEFAULT TRUE
		)`,
		`CREATE TABLE IF NOT EXISTS admin_openvpn_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			prop_autologin BOOLEAN NOT NULL DEFAULT FALSE,
			prop_deny BOOLEAN NOT NULL DEFAULT FALSE,
			group_subnets JSONB NOT NULL DEFAULT '[]'::jsonb,
			access_from JSONB NOT NULL DEFAULT '[]'::jsonb,
			access_to JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS admin_openvpn_users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL DEFAULT '',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_login_at TIMESTAMPTZ,
			status TEXT NOT NULL DEFAULT 'inactive',
			assigned_ip TEXT NOT NULL DEFAULT '',
			group_name TEXT NOT NULL DEFAULT '',
			prop_autologin BOOLEAN NOT NULL DEFAULT FALSE,
			prop_admin BOOLEAN NOT NULL DEFAULT FALSE,
			prop_deny BOOLEAN NOT NULL DEFAULT FALSE,
			prop_autogenerate BOOLEAN NOT NULL DEFAULT TRUE,
			access_from JSONB NOT NULL DEFAULT '[]'::jsonb,
			access_to JSONB NOT NULL DEFAULT '[]'::jsonb,
			mfa_secret TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS admin_openvpn_connections (
			id TEXT PRIMARY KEY,
			client_id TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL,
			real_address TEXT NOT NULL,
			virtual_address TEXT NOT NULL,
			bytes_received BIGINT NOT NULL DEFAULT 0,
			bytes_sent BIGINT NOT NULL DEFAULT 0,
			connected_since TIMESTAMPTZ NOT NULL,
			disconnected_at TIMESTAMPTZ,
			active BOOLEAN NOT NULL DEFAULT TRUE
		)`,
		`CREATE TABLE IF NOT EXISTS admin_openvpn_configs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			username TEXT NOT NULL,
			platform TEXT NOT NULL DEFAULT 'windows',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ,
			download_count INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			content TEXT NOT NULL,
			qr_code TEXT NOT NULL DEFAULT '',
			last_downloaded_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS admin_audit_logs (
			id TEXT PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			actor TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			action TEXT NOT NULL,
			action_type TEXT NOT NULL,
			target TEXT NOT NULL,
			target_type TEXT NOT NULL,
			source TEXT NOT NULL,
			status TEXT NOT NULL,
			ip_address TEXT NOT NULL DEFAULT '',
			details TEXT NOT NULL DEFAULT ''
		)`,
	}
}

func (s *AdminPortalStore) seed(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return nil
	}

	var existing int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_keycloak_users`).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	keycloakGroups := []struct {
		id       string
		name     string
		path     string
		parentID *string
		roles    []string
	}{
		{id: "kcg_admins", name: "admins", path: "/admins", roles: []string{"admin"}},
		{id: "kcg_devs", name: "developers", path: "/developers", roles: []string{"developer"}},
		{id: "kcg_frontend", name: "frontend", path: "/developers/frontend", parentID: stringPointer("kcg_devs"), roles: []string{"developer"}},
		{id: "kcg_support", name: "support", path: "/support", roles: []string{"support"}},
	}
	for _, group := range keycloakGroups {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_keycloak_groups (id, name, path, parent_id, roles, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			group.id, group.name, group.path, group.parentID, toJSON(group.roles), now); err != nil {
			return err
		}
	}

	keycloakRoles := []struct {
		id          string
		name        string
		description string
		composite   bool
	}{
		{id: "kcr_admin", name: "admin", description: "Full administrator access", composite: true},
		{id: "kcr_developer", name: "developer", description: "Developer access"},
		{id: "kcr_user", name: "user", description: "Basic user access"},
		{id: "kcr_support", name: "support", description: "Support access"},
	}
	for _, role := range keycloakRoles {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_keycloak_roles (id, name, description, composite, client_role, container_id, created_at) VALUES ($1,$2,$3,$4,FALSE,'master',$5)`,
			role.id, role.name, role.description, role.composite, now); err != nil {
			return err
		}
	}

	keycloakUsers := []struct {
		id            string
		username      string
		email         string
		firstName     string
		lastName      string
		enabled       bool
		emailVerified bool
		createdAt     time.Time
		groups        []string
		roles         []string
	}{
		{id: "kcu_admin", username: "admin", email: "admin@company.com", firstName: "System", lastName: "Administrator", enabled: true, emailVerified: true, createdAt: now.AddDate(0, -3, 0), groups: []string{"admins", "developers"}, roles: []string{"admin", "user"}},
		{id: "kcu_john", username: "john.doe", email: "john.doe@company.com", firstName: "John", lastName: "Doe", enabled: true, emailVerified: true, createdAt: now.AddDate(0, -2, 0), groups: []string{"developers", "frontend"}, roles: []string{"developer", "user"}},
		{id: "kcu_jane", username: "jane.smith", email: "jane.smith@company.com", firstName: "Jane", lastName: "Smith", enabled: true, emailVerified: true, createdAt: now.AddDate(0, -1, -12), groups: []string{"developers"}, roles: []string{"developer", "user"}},
		{id: "kcu_mike", username: "mike.wilson", email: "mike.wilson@company.com", firstName: "Mike", lastName: "Wilson", enabled: true, emailVerified: false, createdAt: now.AddDate(0, -1, 0), groups: []string{"support"}, roles: []string{"support", "user"}},
		{id: "kcu_sarah", username: "sarah.johnson", email: "sarah.johnson@company.com", firstName: "Sarah", lastName: "Johnson", enabled: false, emailVerified: true, createdAt: now.AddDate(0, -4, 0), groups: []string{}, roles: []string{"user"}},
	}
	for _, user := range keycloakUsers {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_keycloak_users (id, username, email, first_name, last_name, enabled, email_verified, created_at, groups, roles, attributes, required_actions, password_hash, password_temporary) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'{}'::jsonb,'[]'::jsonb,'',FALSE)`,
			user.id, user.username, user.email, user.firstName, user.lastName, user.enabled, user.emailVerified, user.createdAt, toJSON(user.groups), toJSON(user.roles)); err != nil {
			return err
		}
	}

	keycloakSessions := []struct {
		id         string
		username   string
		userID     string
		ipAddress  string
		startAt    time.Time
		lastAccess time.Time
		clients    map[string]string
	}{
		{id: "kcs_1", username: "john.doe", userID: "kcu_john", ipAddress: "192.168.1.100", startAt: now.Add(-2 * time.Hour), lastAccess: now.Add(-30 * time.Second), clients: map[string]string{"portal": "admin-portal", "api": "api-gateway"}},
		{id: "kcs_2", username: "jane.smith", userID: "kcu_jane", ipAddress: "192.168.1.101", startAt: now.Add(-15 * time.Minute), lastAccess: now.Add(-2 * time.Minute), clients: map[string]string{"portal": "admin-portal"}},
		{id: "kcs_3", username: "admin", userID: "kcu_admin", ipAddress: "10.0.0.1", startAt: now.Add(-1 * time.Hour), lastAccess: now.Add(-5 * time.Minute), clients: map[string]string{"portal": "admin-portal", "ops": "monitoring"}},
	}
	for _, session := range keycloakSessions {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_keycloak_sessions (id, username, user_id, ip_address, start_at, last_access_at, clients, active) VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE)`,
			session.id, session.username, session.userID, session.ipAddress, session.startAt, session.lastAccess, toJSON(session.clients)); err != nil {
			return err
		}
	}

	openvpnGroups := []struct {
		id            string
		name          string
		description   string
		propAutologin bool
	}{
		{id: "ovg_devs", name: "developers", description: "Development team VPN access", propAutologin: true},
		{id: "ovg_admins", name: "admins", description: "System administrators", propAutologin: true},
		{id: "ovg_support", name: "support", description: "Support engineers", propAutologin: false},
	}
	for _, group := range openvpnGroups {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_openvpn_groups (id, name, description, prop_autologin, prop_deny, group_subnets, access_from, access_to, created_at) VALUES ($1,$2,$3,$4,FALSE,'[]'::jsonb,'[]'::jsonb,'[]'::jsonb,$5)`,
			group.id, group.name, group.description, group.propAutologin, now); err != nil {
			return err
		}
	}

	openvpnUsers := []struct {
		id            string
		username      string
		email         string
		enabled       bool
		createdAt     time.Time
		lastLoginAt   *time.Time
		status        string
		assignedIP    string
		groupName     string
		propAutologin bool
		propAdmin     bool
	}{
		{id: "ovu_admin", username: "admin", email: "admin@company.com", enabled: true, createdAt: now.AddDate(0, -4, 0), lastLoginAt: timePointer(now.Add(-1 * time.Hour)), status: "active", assignedIP: "10.8.0.1", groupName: "admins", propAutologin: true, propAdmin: true},
		{id: "ovu_john", username: "john.doe", email: "john.doe@company.com", enabled: true, createdAt: now.AddDate(0, -2, 0), lastLoginAt: timePointer(now.Add(-2 * time.Hour)), status: "active", assignedIP: "10.8.0.2", groupName: "developers", propAutologin: true},
		{id: "ovu_jane", username: "jane.smith", email: "jane.smith@company.com", enabled: true, createdAt: now.AddDate(0, -1, -15), lastLoginAt: timePointer(now.Add(-30 * time.Minute)), status: "active", assignedIP: "10.8.0.3", groupName: "developers", propAutologin: true},
		{id: "ovu_mike", username: "mike.wilson", email: "mike.wilson@company.com", enabled: true, createdAt: now.AddDate(0, -1, 0), status: "inactive", assignedIP: "10.8.0.4", groupName: "support"},
	}
	for _, user := range openvpnUsers {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_openvpn_users (id, username, email, enabled, created_at, last_login_at, status, assigned_ip, group_name, prop_autologin, prop_admin, prop_deny, prop_autogenerate, access_from, access_to, mfa_secret) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,FALSE,TRUE,'[]'::jsonb,'[]'::jsonb,'')`,
			user.id, user.username, user.email, user.enabled, user.createdAt, user.lastLoginAt, user.status, user.assignedIP, user.groupName, user.propAutologin, user.propAdmin); err != nil {
			return err
		}
	}

	openvpnConnections := []struct {
		id             string
		clientID       string
		username       string
		realAddress    string
		virtualIP      string
		bytesReceived  int64
		bytesSent      int64
		connectedSince time.Time
	}{
		{id: "ovc_1", clientID: "client-john", username: "john.doe", realAddress: "203.0.113.45:51234", virtualIP: "10.8.0.2", bytesReceived: 15234567, bytesSent: 8765432, connectedSince: now.Add(-2 * time.Hour)},
		{id: "ovc_2", clientID: "client-jane", username: "jane.smith", realAddress: "198.51.100.78:48765", virtualIP: "10.8.0.3", bytesReceived: 8976543, bytesSent: 4321098, connectedSince: now.Add(-30 * time.Minute)},
		{id: "ovc_3", clientID: "client-admin", username: "admin", realAddress: "192.0.2.100:52341", virtualIP: "10.8.0.1", bytesReceived: 45678901, bytesSent: 23456789, connectedSince: now.Add(-4 * time.Hour)},
	}
	for _, conn := range openvpnConnections {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_openvpn_connections (id, client_id, username, real_address, virtual_address, bytes_received, bytes_sent, connected_since, active) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,TRUE)`,
			conn.id, conn.clientID, conn.username, conn.realAddress, conn.virtualIP, conn.bytesReceived, conn.bytesSent, conn.connectedSince); err != nil {
			return err
		}
	}

	configs := []struct {
		id        string
		name      string
		username  string
		createdAt time.Time
		status    string
		content   string
	}{
		{id: "ovcfg_1", name: "john.doe-windows", username: "john.doe", createdAt: now.AddDate(0, 0, -30), status: "active", content: sampleVPNConfig("john.doe")},
		{id: "ovcfg_2", name: "jane.smith-linux", username: "jane.smith", createdAt: now.AddDate(0, 0, -20), status: "active", content: sampleVPNConfig("jane.smith")},
		{id: "ovcfg_3", name: "admin-mobile", username: "admin", createdAt: now.AddDate(0, 0, -10), status: "active", content: sampleVPNConfig("admin")},
	}
	for _, cfg := range configs {
		qr := buildSVGDataURI(cfg.name)
		if _, err := tx.Exec(ctx, `INSERT INTO admin_openvpn_configs (id, name, username, platform, created_at, download_count, status, content, qr_code) VALUES ($1,$2,$3,'windows',$4,0,$5,$6,$7)`,
			cfg.id, cfg.name, cfg.username, cfg.createdAt, cfg.status, cfg.content, qr); err != nil {
			return err
		}
	}

	auditEntries := []application.AuditLog{
		{ID: "audit_1", Timestamp: now.Add(-5 * time.Minute), Actor: "admin", ActorType: "admin", Action: "User Created", ActionType: "create", Target: "john.doe", TargetType: "user", Source: "keycloak", Status: "success", IPAddress: "192.168.1.100", Details: "New user account created"},
		{ID: "audit_2", Timestamp: now.Add(-12 * time.Minute), Actor: "admin", ActorType: "admin", Action: "Password Reset", ActionType: "update", Target: "jane.smith", TargetType: "user", Source: "keycloak", Status: "success", IPAddress: "192.168.1.100", Details: "Admin reset password"},
		{ID: "audit_3", Timestamp: now.Add(-25 * time.Minute), Actor: "system", ActorType: "system", Action: "VPN Session Started", ActionType: "login", Target: "mike.wilson", TargetType: "vpn", Source: "openvpn", Status: "success", IPAddress: "203.0.113.45", Details: "VPN connection established"},
		{ID: "audit_4", Timestamp: now.Add(-40 * time.Minute), Actor: "admin", ActorType: "admin", Action: "User Added to Group", ActionType: "assign", Target: "sarah.johnson", TargetType: "group", Source: "keycloak", Status: "success", IPAddress: "192.168.1.100", Details: "Added user to developers"},
	}
	for _, entry := range auditEntries {
		if _, err := tx.Exec(ctx, `INSERT INTO admin_audit_logs (id, timestamp, actor, actor_type, action, action_type, target, target_type, source, status, ip_address, details) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			entry.ID, entry.Timestamp, entry.Actor, entry.ActorType, entry.Action, entry.ActionType, entry.Target, entry.TargetType, entry.Source, entry.Status, entry.IPAddress, entry.Details); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func stringPointer(value string) *string {
	return &value
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func toJSON(value any) []byte {
	out, _ := json.Marshal(value)
	return out
}

func sampleVPNConfig(username string) string {
	return fmt.Sprintf("client\ndev tun\nproto udp\nremote vpn.example.com 1194\nresolv-retry infinite\nnobind\npersist-key\npersist-tun\ncipher AES-256-CBC\nauth SHA256\nverb 3\n# sample profile for %s\n", username)
}

func buildSVGDataURI(label string) string {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="240" height="240"><rect width="100%%" height="100%%" fill="#ffffff"/><rect x="20" y="20" width="200" height="200" rx="18" fill="#0f766e"/><text x="120" y="112" text-anchor="middle" fill="#ffffff" font-size="18" font-family="Arial">VPN QR</text><text x="120" y="140" text-anchor="middle" fill="#ccfbf1" font-size="12" font-family="Arial">%s</text></svg>`, htmlEscape(label))
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return replacer.Replace(value)
}

func (s *AdminPortalStore) GetDashboardSummary(ctx context.Context, recentLimit int) (application.DashboardSummary, error) {
	if s == nil || s.pool == nil {
		return application.DashboardSummary{}, domainerr.New(domainerr.CodePreconditionFail, "database is not configured")
	}

	var summary application.DashboardSummary
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_keycloak_users`).Scan(&summary.TotalUsers); err != nil {
		return summary, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_keycloak_sessions WHERE active = TRUE`).Scan(&summary.ActiveSessions); err != nil {
		return summary, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_openvpn_connections WHERE active = TRUE`).Scan(&summary.VPNConnections); err != nil {
		return summary, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_openvpn_users`).Scan(&summary.TotalVPNUsers); err != nil {
		return summary, err
	}
	items, _, err := s.ListAuditLogs(ctx, application.AuditLogFilter{Limit: recentLimit})
	if err != nil {
		return summary, err
	}
	summary.RecentActivity = items
	return summary, nil
}

func (s *AdminPortalStore) ListAuditLogs(ctx context.Context, filter application.AuditLogFilter) ([]application.AuditLog, int, error) {
	if s == nil || s.pool == nil {
		return nil, 0, domainerr.New(domainerr.CodePreconditionFail, "database is not configured")
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	where := []string{"1=1"}
	args := []any{}
	appendArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if q := strings.TrimSpace(filter.Search); q != "" {
		placeholder := appendArg("%" + q + "%")
		where = append(where, fmt.Sprintf("(actor ILIKE %s OR action ILIKE %s OR target ILIKE %s OR details ILIKE %s OR ip_address ILIKE %s)", placeholder, placeholder, placeholder, placeholder, placeholder))
	}
	if source := strings.TrimSpace(filter.Source); source != "" && source != "all" {
		where = append(where, fmt.Sprintf("source = %s", appendArg(source)))
	}
	if status := strings.TrimSpace(filter.Status); status != "" && status != "all" {
		where = append(where, fmt.Sprintf("status = %s", appendArg(status)))
	}
	if actionType := strings.TrimSpace(filter.ActionType); actionType != "" && actionType != "all" {
		where = append(where, fmt.Sprintf("action_type = %s", appendArg(actionType)))
	}

	whereClause := strings.Join(where, " AND ")
	var total int
	if err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM admin_audit_logs WHERE %s`, whereClause), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, filter.Limit, filter.Offset)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT id, timestamp, actor, actor_type, action, action_type, target, target_type, source, status, ip_address, details FROM admin_audit_logs WHERE %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d`, whereClause, len(queryArgs)-1, len(queryArgs)), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]application.AuditLog, 0)
	for rows.Next() {
		var item application.AuditLog
		if err := rows.Scan(&item.ID, &item.Timestamp, &item.Actor, &item.ActorType, &item.Action, &item.ActionType, &item.Target, &item.TargetType, &item.Source, &item.Status, &item.IPAddress, &item.Details); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *AdminPortalStore) InsertAuditLog(ctx context.Context, entry application.AuditLog) error {
	if s == nil || s.pool == nil {
		return domainerr.New(domainerr.CodePreconditionFail, "database is not configured")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_audit_logs (id, timestamp, actor, actor_type, action, action_type, target, target_type, source, status, ip_address, details) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		entry.ID, entry.Timestamp, entry.Actor, entry.ActorType, entry.Action, entry.ActionType, entry.Target, entry.TargetType, entry.Source, entry.Status, entry.IPAddress, entry.Details)
	return err
}

func decodeStringSlice(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out
}

func decodeStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	var out map[string]string
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return map[string]string{}
	}
	return out
}

func stringSliceContains(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func upsertString(items []string, value string) []string {
	if stringSliceContains(items, value) {
		return items
	}
	return append(items, value)
}

func removeString(items []string, value string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(item, value) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func mapPQLError(err error, duplicateTarget string) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return domainerr.New(domainerr.CodeConflict, duplicateTarget+" already exists")
	case "23503":
		return domainerr.New(domainerr.CodeNotFound, "referenced record not found")
	default:
		return err
	}
}

func (s *AdminPortalStore) ListKeycloakUsers(ctx context.Context, search string, first, max int) ([]application.KeycloakUserView, int, error) {
	where := "1=1"
	args := []any{}
	if q := strings.TrimSpace(search); q != "" {
		where = `username ILIKE $1 OR email ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1`
		args = append(args, "%"+q+"%")
	}

	var total int
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM admin_keycloak_users WHERE %s`, where)
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, max, first)
	sql := fmt.Sprintf(`SELECT id, username, email, first_name, last_name, enabled, email_verified, created_at, groups, roles FROM admin_keycloak_users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, len(queryArgs)-1, len(queryArgs))
	rows, err := s.pool.Query(ctx, sql, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]application.KeycloakUserView, 0)
	for rows.Next() {
		var item application.KeycloakUserView
		var createdAt time.Time
		var groupsRaw, rolesRaw []byte
		if err := rows.Scan(&item.ID, &item.Username, &item.Email, &item.FirstName, &item.LastName, &item.Enabled, &item.EmailVerified, &createdAt, &groupsRaw, &rolesRaw); err != nil {
			return nil, 0, err
		}
		item.CreatedTimestamp = createdAt.UnixMilli()
		item.Groups = decodeStringSlice(groupsRaw)
		item.Roles = decodeStringSlice(rolesRaw)
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (s *AdminPortalStore) CreateKeycloakUser(ctx context.Context, input application.KeycloakUserCreateInput) (application.KeycloakUserView, error) {
	id := "kcu_" + strings.ToLower(strings.ReplaceAll(input.Username, ".", "_"))
	if input.Password == "" {
		input.Password = input.Username + "@123"
	}
	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_keycloak_users (id, username, email, first_name, last_name, enabled, email_verified, created_at, groups, roles, attributes, required_actions, password_hash, password_temporary) VALUES ($1,$2,$3,$4,$5,$6,FALSE,$7,'[]'::jsonb,'[]'::jsonb,'{}'::jsonb,'[]'::jsonb,$8,$9)`,
		id, input.Username, input.Email, input.FirstName, input.LastName, input.Enabled, now, applicationHashPassword(input.Password), input.TemporaryPassword)
	if err != nil {
		return application.KeycloakUserView{}, mapPQLError(err, "user")
	}
	return application.KeycloakUserView{
		ID:               id,
		Username:         input.Username,
		Email:            input.Email,
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		Enabled:          input.Enabled,
		EmailVerified:    false,
		CreatedTimestamp: now.UnixMilli(),
		Groups:           []string{},
		Roles:            []string{},
	}, nil
}

func (s *AdminPortalStore) UpdateKeycloakUser(ctx context.Context, id string, input application.KeycloakUserUpdateInput) (application.KeycloakUserView, error) {
	user, createdAt, groups, roles, err := s.getKeycloakUser(ctx, id)
	if err != nil {
		return application.KeycloakUserView{}, err
	}
	if input.Email != nil {
		user.Email = strings.TrimSpace(*input.Email)
	}
	if input.FirstName != nil {
		user.FirstName = strings.TrimSpace(*input.FirstName)
	}
	if input.LastName != nil {
		user.LastName = strings.TrimSpace(*input.LastName)
	}
	if input.Enabled != nil {
		user.Enabled = *input.Enabled
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET email=$2, first_name=$3, last_name=$4, enabled=$5 WHERE id=$1`, id, user.Email, user.FirstName, user.LastName, user.Enabled)
	if err != nil {
		return application.KeycloakUserView{}, err
	}
	user.CreatedTimestamp = createdAt.UnixMilli()
	user.Groups = groups
	user.Roles = roles
	return user, nil
}

func (s *AdminPortalStore) DeleteKeycloakUser(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM admin_keycloak_users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "user not found")
	}
	_, _ = s.pool.Exec(ctx, `UPDATE admin_keycloak_sessions SET active = FALSE WHERE user_id = $1`, id)
	return nil
}

func (s *AdminPortalStore) SetKeycloakUserPassword(ctx context.Context, id, passwordHash string, temporary bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET password_hash=$2, password_temporary=$3 WHERE id=$1`, id, passwordHash, temporary)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "user not found")
	}
	return nil
}

func (s *AdminPortalStore) GetKeycloakUserRoles(ctx context.Context, id string) ([]application.KeycloakRoleView, error) {
	_, _, _, roles, err := s.getKeycloakUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.rolesByNames(ctx, roles)
}

func (s *AdminPortalStore) AssignRolesToKeycloakUser(ctx context.Context, userID string, roles []application.RoleRef) error {
	user, _, _, currentRoles, err := s.getKeycloakUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if name := strings.TrimSpace(role.Name); name != "" {
			currentRoles = upsertString(currentRoles, name)
		}
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET roles=$2 WHERE id=$1`, user.ID, toJSON(currentRoles))
	return err
}

func (s *AdminPortalStore) RemoveRolesFromKeycloakUser(ctx context.Context, userID string, roles []application.RoleRef) error {
	user, _, _, currentRoles, err := s.getKeycloakUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, role := range roles {
		currentRoles = removeString(currentRoles, role.Name)
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET roles=$2 WHERE id=$1`, user.ID, toJSON(currentRoles))
	return err
}

func (s *AdminPortalStore) GetKeycloakUserGroups(ctx context.Context, id string) ([]application.KeycloakGroupView, error) {
	_, _, groups, _, err := s.getKeycloakUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.groupsByNames(ctx, groups)
}

func (s *AdminPortalStore) AddKeycloakUserToGroup(ctx context.Context, userID, groupID string) error {
	user, _, groups, roles, err := s.getKeycloakUser(ctx, userID)
	if err != nil {
		return err
	}
	group, err := s.getKeycloakGroupRow(ctx, groupID)
	if err != nil {
		return err
	}
	groups = upsertString(groups, group.Name)
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET groups=$2, roles=$3 WHERE id=$1`, user.ID, toJSON(groups), toJSON(roles))
	return err
}

func (s *AdminPortalStore) RemoveKeycloakUserFromGroup(ctx context.Context, userID, groupID string) error {
	user, _, groups, roles, err := s.getKeycloakUser(ctx, userID)
	if err != nil {
		return err
	}
	group, err := s.getKeycloakGroupRow(ctx, groupID)
	if err != nil {
		return err
	}
	groups = removeString(groups, group.Name)
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET groups=$2, roles=$3 WHERE id=$1`, user.ID, toJSON(groups), toJSON(roles))
	return err
}

func (s *AdminPortalStore) getKeycloakUser(ctx context.Context, id string) (application.KeycloakUserView, time.Time, []string, []string, error) {
	var user application.KeycloakUserView
	var createdAt time.Time
	var groupsRaw, rolesRaw []byte
	err := s.pool.QueryRow(ctx, `SELECT id, username, email, first_name, last_name, enabled, email_verified, created_at, groups, roles FROM admin_keycloak_users WHERE id = $1`, id).
		Scan(&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName, &user.Enabled, &user.EmailVerified, &createdAt, &groupsRaw, &rolesRaw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return application.KeycloakUserView{}, time.Time{}, nil, nil, domainerr.New(domainerr.CodeNotFound, "user not found")
		}
		return application.KeycloakUserView{}, time.Time{}, nil, nil, err
	}
	groups := decodeStringSlice(groupsRaw)
	roles := decodeStringSlice(rolesRaw)
	user.CreatedTimestamp = createdAt.UnixMilli()
	user.Groups = groups
	user.Roles = roles
	return user, createdAt, groups, roles, nil
}

func (s *AdminPortalStore) ListKeycloakGroups(ctx context.Context) ([]application.KeycloakGroupView, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, path, parent_id FROM admin_keycloak_groups ORDER BY path ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		ID       string
		Name     string
		Path     string
		ParentID *string
	}
	all := make([]row, 0)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.ID, &item.Name, &item.Path, &item.ParentID); err != nil {
			return nil, err
		}
		all = append(all, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	children := map[string][]application.KeycloakGroupView{}
	roots := make([]application.KeycloakGroupView, 0)
	idToGroup := map[string]application.KeycloakGroupView{}
	for _, item := range all {
		idToGroup[item.ID] = application.KeycloakGroupView{ID: item.ID, Name: item.Name, Path: item.Path}
	}
	for i := len(all) - 1; i >= 0; i-- {
		item := all[i]
		group := idToGroup[item.ID]
		group.SubGroups = children[item.ID]
		if item.ParentID == nil || strings.TrimSpace(*item.ParentID) == "" {
			roots = append([]application.KeycloakGroupView{group}, roots...)
			continue
		}
		parentID := strings.TrimSpace(*item.ParentID)
		children[parentID] = append([]application.KeycloakGroupView{group}, children[parentID]...)
	}
	return roots, nil
}

func (s *AdminPortalStore) CreateKeycloakGroup(ctx context.Context, name string, parentID *string) (application.KeycloakGroupView, error) {
	id := "kcg_" + strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	path := "/" + name
	if parentID != nil && strings.TrimSpace(*parentID) != "" {
		parent, err := s.getKeycloakGroupRow(ctx, *parentID)
		if err != nil {
			return application.KeycloakGroupView{}, err
		}
		path = strings.TrimRight(parent.Path, "/") + "/" + name
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_keycloak_groups (id, name, path, parent_id, roles, created_at) VALUES ($1,$2,$3,$4,'[]'::jsonb,$5)`,
		id, name, path, parentID, time.Now().UTC())
	if err != nil {
		return application.KeycloakGroupView{}, mapPQLError(err, "group")
	}
	return application.KeycloakGroupView{ID: id, Name: name, Path: path}, nil
}

func (s *AdminPortalStore) UpdateKeycloakGroup(ctx context.Context, id, name string) (application.KeycloakGroupView, error) {
	group, err := s.getKeycloakGroupRow(ctx, id)
	if err != nil {
		return application.KeycloakGroupView{}, err
	}
	oldName := group.Name
	oldPath := group.Path
	newPath := "/" + name
	if group.ParentID != nil && strings.TrimSpace(*group.ParentID) != "" {
		parent, err := s.getKeycloakGroupRow(ctx, *group.ParentID)
		if err != nil {
			return application.KeycloakGroupView{}, err
		}
		newPath = strings.TrimRight(parent.Path, "/") + "/" + name
	}
	if _, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_groups SET name=$2, path=$3 WHERE id=$1`, id, name, newPath); err != nil {
		return application.KeycloakGroupView{}, mapPQLError(err, "group")
	}
	if _, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_groups SET path = regexp_replace(path, '^' || $2, $3) WHERE path LIKE $2 || '/%'`, id, oldPath, newPath); err != nil {
		return application.KeycloakGroupView{}, err
	}
	if err := s.renameGroupForUsers(ctx, oldName, name); err != nil {
		return application.KeycloakGroupView{}, err
	}
	return application.KeycloakGroupView{ID: id, Name: name, Path: newPath}, nil
}

func (s *AdminPortalStore) DeleteKeycloakGroup(ctx context.Context, id string) error {
	group, err := s.getKeycloakGroupRow(ctx, id)
	if err != nil {
		return err
	}
	names := []string{group.Name}
	rows, err := s.pool.Query(ctx, `SELECT name FROM admin_keycloak_groups WHERE path LIKE $1 || '/%'`, group.Path)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		names = append(names, name)
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM admin_keycloak_groups WHERE id = $1`, id); err != nil {
		return err
	}
	for _, name := range names {
		if err := s.renameGroupForUsers(ctx, name, ""); err != nil {
			return err
		}
	}
	return nil
}

func (s *AdminPortalStore) GetKeycloakGroupMembers(ctx context.Context, id string) ([]application.KeycloakUserView, error) {
	group, err := s.getKeycloakGroupRow(ctx, id)
	if err != nil {
		return nil, err
	}
	users, _, err := s.ListKeycloakUsers(ctx, "", 0, 500)
	if err != nil {
		return nil, err
	}
	out := make([]application.KeycloakUserView, 0)
	for _, user := range users {
		if stringSliceContains(user.Groups, group.Name) {
			out = append(out, user)
		}
	}
	return out, nil
}

func (s *AdminPortalStore) GetKeycloakGroupRoles(ctx context.Context, id string) ([]application.KeycloakRoleView, error) {
	group, err := s.getKeycloakGroupRow(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.rolesByNames(ctx, group.Roles)
}

func (s *AdminPortalStore) AssignRolesToKeycloakGroup(ctx context.Context, groupID string, roles []application.RoleRef) error {
	group, err := s.getKeycloakGroupRow(ctx, groupID)
	if err != nil {
		return err
	}
	current := append([]string(nil), group.Roles...)
	for _, role := range roles {
		current = upsertString(current, role.Name)
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_groups SET roles = $2 WHERE id = $1`, groupID, toJSON(current))
	return err
}

func (s *AdminPortalStore) RemoveRolesFromKeycloakGroup(ctx context.Context, groupID string, roles []application.RoleRef) error {
	group, err := s.getKeycloakGroupRow(ctx, groupID)
	if err != nil {
		return err
	}
	current := append([]string(nil), group.Roles...)
	for _, role := range roles {
		current = removeString(current, role.Name)
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_keycloak_groups SET roles = $2 WHERE id = $1`, groupID, toJSON(current))
	return err
}

func (s *AdminPortalStore) ListKeycloakRoles(ctx context.Context) ([]application.KeycloakRoleView, error) {
	counts, err := s.roleUsageCounts(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, description, composite, client_role, container_id FROM admin_keycloak_roles ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.KeycloakRoleView, 0)
	for rows.Next() {
		var item application.KeycloakRoleView
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Composite, &item.ClientRole, &item.ContainerID); err != nil {
			return nil, err
		}
		item.UserCount = counts[strings.ToLower(item.Name)]
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AdminPortalStore) CreateKeycloakRole(ctx context.Context, name, description string) (application.KeycloakRoleView, error) {
	id := "kcr_" + strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_keycloak_roles (id, name, description, composite, client_role, container_id, created_at) VALUES ($1,$2,$3,FALSE,FALSE,'master',$4)`,
		id, name, description, time.Now().UTC())
	if err != nil {
		return application.KeycloakRoleView{}, mapPQLError(err, "role")
	}
	return application.KeycloakRoleView{ID: id, Name: name, Description: description, ContainerID: "master"}, nil
}

func (s *AdminPortalStore) DeleteKeycloakRole(ctx context.Context, name string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM admin_keycloak_roles WHERE name = $1`, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "role not found")
	}
	if err := s.removeRoleEverywhere(ctx, name); err != nil {
		return err
	}
	return nil
}

func (s *AdminPortalStore) ListKeycloakSessions(ctx context.Context) ([]application.KeycloakSessionView, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, user_id, ip_address, start_at, last_access_at, clients FROM admin_keycloak_sessions WHERE active = TRUE ORDER BY last_access_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.KeycloakSessionView, 0)
	for rows.Next() {
		var item application.KeycloakSessionView
		var startAt, lastAccess time.Time
		var clientsRaw []byte
		if err := rows.Scan(&item.ID, &item.Username, &item.UserID, &item.IPAddress, &startAt, &lastAccess, &clientsRaw); err != nil {
			return nil, err
		}
		item.Start = startAt.UnixMilli()
		item.LastAccess = lastAccess.UnixMilli()
		item.Clients = decodeStringMap(clientsRaw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AdminPortalStore) LogoutKeycloakSession(ctx context.Context, sessionID string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_sessions SET active = FALSE WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "session not found")
	}
	return nil
}

func (s *AdminPortalStore) LogoutKeycloakUser(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_sessions SET active = FALSE WHERE user_id = $1`, userID)
	return err
}

func (s *AdminPortalStore) LogoutAllKeycloakSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_sessions SET active = FALSE WHERE active = TRUE`)
	return err
}

type keycloakGroupRow struct {
	ID       string
	Name     string
	Path     string
	ParentID *string
	Roles    []string
}

func (s *AdminPortalStore) getKeycloakGroupRow(ctx context.Context, id string) (keycloakGroupRow, error) {
	var row keycloakGroupRow
	var rolesRaw []byte
	err := s.pool.QueryRow(ctx, `SELECT id, name, path, parent_id, roles FROM admin_keycloak_groups WHERE id = $1`, id).
		Scan(&row.ID, &row.Name, &row.Path, &row.ParentID, &rolesRaw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return keycloakGroupRow{}, domainerr.New(domainerr.CodeNotFound, "group not found")
		}
		return keycloakGroupRow{}, err
	}
	row.Roles = decodeStringSlice(rolesRaw)
	return row, nil
}

func (s *AdminPortalStore) renameGroupForUsers(ctx context.Context, oldName, newName string) error {
	rows, err := s.pool.Query(ctx, `SELECT id, groups FROM admin_keycloak_users`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type update struct {
		id     string
		groups []string
	}
	updates := make([]update, 0)
	for rows.Next() {
		var id string
		var groupsRaw []byte
		if err := rows.Scan(&id, &groupsRaw); err != nil {
			return err
		}
		groups := decodeStringSlice(groupsRaw)
		changed := false
		next := make([]string, 0, len(groups))
		for _, group := range groups {
			if strings.EqualFold(group, oldName) {
				changed = true
				if strings.TrimSpace(newName) != "" {
					next = append(next, newName)
				}
				continue
			}
			next = append(next, group)
		}
		if changed {
			updates = append(updates, update{id: id, groups: next})
		}
	}
	for _, item := range updates {
		if _, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET groups = $2 WHERE id = $1`, item.id, toJSON(item.groups)); err != nil {
			return err
		}
	}
	return nil
}

func (s *AdminPortalStore) rolesByNames(ctx context.Context, names []string) ([]application.KeycloakRoleView, error) {
	if len(names) == 0 {
		return []application.KeycloakRoleView{}, nil
	}
	all, err := s.ListKeycloakRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]application.KeycloakRoleView, 0)
	for _, role := range all {
		if stringSliceContains(names, role.Name) {
			out = append(out, role)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *AdminPortalStore) groupsByNames(ctx context.Context, names []string) ([]application.KeycloakGroupView, error) {
	if len(names) == 0 {
		return []application.KeycloakGroupView{}, nil
	}
	all, err := s.ListKeycloakGroups(ctx)
	if err != nil {
		return nil, err
	}
	flat := flattenGroups(all)
	out := make([]application.KeycloakGroupView, 0)
	for _, group := range flat {
		if stringSliceContains(names, group.Name) {
			out = append(out, application.KeycloakGroupView{ID: group.ID, Name: group.Name, Path: group.Path})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func flattenGroups(items []application.KeycloakGroupView) []application.KeycloakGroupView {
	out := make([]application.KeycloakGroupView, 0)
	for _, item := range items {
		out = append(out, application.KeycloakGroupView{ID: item.ID, Name: item.Name, Path: item.Path})
		if len(item.SubGroups) > 0 {
			out = append(out, flattenGroups(item.SubGroups)...)
		}
	}
	return out
}

func (s *AdminPortalStore) roleUsageCounts(ctx context.Context) (map[string]int, error) {
	counts := map[string]int{}
	rows, err := s.pool.Query(ctx, `SELECT roles FROM admin_keycloak_users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		for _, role := range decodeStringSlice(raw) {
			counts[strings.ToLower(role)]++
		}
	}
	return counts, rows.Err()
}

func (s *AdminPortalStore) removeRoleEverywhere(ctx context.Context, roleName string) error {
	userRows, err := s.pool.Query(ctx, `SELECT id, roles FROM admin_keycloak_users`)
	if err != nil {
		return err
	}
	defer userRows.Close()
	type roleUpdate struct {
		id    string
		roles []string
	}
	userUpdates := make([]roleUpdate, 0)
	for userRows.Next() {
		var id string
		var raw []byte
		if err := userRows.Scan(&id, &raw); err != nil {
			return err
		}
		roles := removeString(decodeStringSlice(raw), roleName)
		userUpdates = append(userUpdates, roleUpdate{id: id, roles: roles})
	}
	for _, item := range userUpdates {
		if _, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_users SET roles = $2 WHERE id = $1`, item.id, toJSON(item.roles)); err != nil {
			return err
		}
	}

	groupRows, err := s.pool.Query(ctx, `SELECT id, roles FROM admin_keycloak_groups`)
	if err != nil {
		return err
	}
	defer groupRows.Close()
	groupUpdates := make([]roleUpdate, 0)
	for groupRows.Next() {
		var id string
		var raw []byte
		if err := groupRows.Scan(&id, &raw); err != nil {
			return err
		}
		roles := removeString(decodeStringSlice(raw), roleName)
		groupUpdates = append(groupUpdates, roleUpdate{id: id, roles: roles})
	}
	for _, item := range groupUpdates {
		if _, err := s.pool.Exec(ctx, `UPDATE admin_keycloak_groups SET roles = $2 WHERE id = $1`, item.id, toJSON(item.roles)); err != nil {
			return err
		}
	}
	return nil
}

func (s *AdminPortalStore) ListOpenVPNUsers(ctx context.Context) ([]application.OpenVPNUserView, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, email, enabled, created_at, last_login_at, status, assigned_ip, group_name, prop_autologin, prop_admin, prop_deny, prop_autogenerate, access_from, access_to FROM admin_openvpn_users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.OpenVPNUserView, 0)
	for rows.Next() {
		var item application.OpenVPNUserView
		var createdAt time.Time
		var lastLogin *time.Time
		var groupName string
		var accessFromRaw, accessToRaw []byte
		if err := rows.Scan(&item.ID, &item.Username, &item.Email, &item.Enabled, &createdAt, &lastLogin, &item.Status, &item.AssignedIP, &groupName, &item.PropAutologin, &item.PropAdmin, &item.PropDeny, &item.PropAutogenerate, &accessFromRaw, &accessToRaw); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		if lastLogin != nil {
			v := lastLogin.Format(time.RFC3339)
			item.LastLogin = &v
		}
		item.Group = groupName
		item.AccessFrom = decodeStringSlice(accessFromRaw)
		item.AccessTo = decodeStringSlice(accessToRaw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AdminPortalStore) CreateOpenVPNUser(ctx context.Context, input application.OpenVPNUserCreateInput) (application.OpenVPNUserView, error) {
	id := "ovu_" + strings.ToLower(strings.ReplaceAll(input.Username, ".", "_"))
	now := time.Now().UTC()
	groupName := ""
	if input.Group != nil {
		groupName = strings.TrimSpace(*input.Group)
	}
	status := "inactive"
	if input.Enabled {
		status = "active"
	}
	assignedIP := fmt.Sprintf("10.8.0.%d", 10+time.Now().Nanosecond()%200)
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_openvpn_users (id, username, email, enabled, created_at, status, assigned_ip, group_name, prop_autologin, prop_admin, prop_deny, prop_autogenerate, access_from, access_to, mfa_secret) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,FALSE,TRUE,'[]'::jsonb,'[]'::jsonb,'')`,
		id, input.Username, input.Email, input.Enabled, now, status, assignedIP, groupName, input.PropAutologin, input.PropAdmin)
	if err != nil {
		return application.OpenVPNUserView{}, mapPQLError(err, "vpn user")
	}
	return application.OpenVPNUserView{ID: id, Username: input.Username, Email: input.Email, Enabled: input.Enabled, CreatedAt: now.Format(time.RFC3339), Status: status, AssignedIP: assignedIP, Group: groupName, PropAutologin: input.PropAutologin, PropAdmin: input.PropAdmin, PropAutogenerate: true}, nil
}

func (s *AdminPortalStore) UpdateOpenVPNUser(ctx context.Context, username string, input application.OpenVPNUserUpdateInput) (application.OpenVPNUserView, error) {
	user, err := s.getOpenVPNUser(ctx, username)
	if err != nil {
		return application.OpenVPNUserView{}, err
	}
	if input.Email != nil {
		user.Email = strings.TrimSpace(*input.Email)
	}
	if input.Enabled != nil {
		user.Enabled = *input.Enabled
		if !user.Enabled {
			user.Status = "suspended"
		}
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_openvpn_users SET email=$2, enabled=$3, status=$4 WHERE username=$1`, username, user.Email, user.Enabled, user.Status)
	if err != nil {
		return application.OpenVPNUserView{}, err
	}
	return user, nil
}

func (s *AdminPortalStore) DeleteOpenVPNUser(ctx context.Context, username string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM admin_openvpn_users WHERE username = $1`, username)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "vpn user not found")
	}
	_, _ = s.pool.Exec(ctx, `UPDATE admin_openvpn_connections SET active = FALSE, disconnected_at = NOW() WHERE username = $1 AND active = TRUE`, username)
	return nil
}

func (s *AdminPortalStore) SetOpenVPNUserProps(ctx context.Context, username string, props application.OpenVPNUserProps) error {
	user, err := s.getOpenVPNUser(ctx, username)
	if err != nil {
		return err
	}
	groupName := user.Group
	if props.Group != nil {
		groupName = strings.TrimSpace(*props.Group)
	}
	status := user.Status
	enabled := user.Enabled
	if props.PropDeny {
		status = "suspended"
		enabled = false
	} else if enabled {
		status = "active"
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_openvpn_users SET group_name=$2, prop_autologin=$3, prop_admin=$4, prop_deny=$5, prop_autogenerate=$6, access_from=$7, access_to=$8, enabled=$9, status=$10 WHERE username=$1`,
		username, groupName, props.PropAutologin, props.PropAdmin, props.PropDeny, props.PropAutogenerate, toJSON(props.AccessFrom), toJSON(props.AccessTo), enabled, status)
	if err != nil {
		return err
	}
	if props.PropDeny {
		_, _ = s.pool.Exec(ctx, `UPDATE admin_openvpn_connections SET active = FALSE, disconnected_at = NOW() WHERE username = $1 AND active = TRUE`, username)
	}
	return nil
}

func (s *AdminPortalStore) GetOpenVPNUserProps(ctx context.Context, username string) (application.OpenVPNUserProps, error) {
	user, err := s.getOpenVPNUser(ctx, username)
	if err != nil {
		return application.OpenVPNUserProps{}, err
	}
	groupName := strings.TrimSpace(user.Group)
	props := application.OpenVPNUserProps{
		PropAutologin:    user.PropAutologin,
		PropAdmin:        user.PropAdmin,
		PropDeny:         user.PropDeny,
		PropAutogenerate: user.PropAutogenerate,
		AccessFrom:       user.AccessFrom,
		AccessTo:         user.AccessTo,
	}
	if groupName != "" {
		props.Group = &groupName
	}
	return props, nil
}

func (s *AdminPortalStore) GenerateOpenVPNUserMFA(ctx context.Context, username string) (string, error) {
	secret := strings.ToUpper(strings.ReplaceAll("mfa-"+username+"-"+fmt.Sprint(time.Now().UnixNano()), ".", ""))
	tag, err := s.pool.Exec(ctx, `UPDATE admin_openvpn_users SET mfa_secret = $2 WHERE username = $1`, username, secret)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", domainerr.New(domainerr.CodeNotFound, "vpn user not found")
	}
	return secret, nil
}

func (s *AdminPortalStore) ListOpenVPNGroups(ctx context.Context) ([]application.OpenVPNGroupView, error) {
	userCounts, err := s.openVPNGroupCounts(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, description, prop_autologin, prop_deny, group_subnets, access_from, access_to FROM admin_openvpn_groups ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.OpenVPNGroupView, 0)
	for rows.Next() {
		var item application.OpenVPNGroupView
		var groupSubnetsRaw, accessFromRaw, accessToRaw []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.PropAutologin, &item.PropDeny, &groupSubnetsRaw, &accessFromRaw, &accessToRaw); err != nil {
			return nil, err
		}
		item.GroupSubnets = decodeStringSlice(groupSubnetsRaw)
		item.AccessFrom = decodeStringSlice(accessFromRaw)
		item.AccessTo = decodeStringSlice(accessToRaw)
		item.UserCount = userCounts[strings.ToLower(item.Name)]
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AdminPortalStore) CreateOpenVPNGroup(ctx context.Context, input application.OpenVPNGroupCreateInput) (application.OpenVPNGroupView, error) {
	id := "ovg_" + strings.ToLower(strings.ReplaceAll(input.Name, " ", "_"))
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_openvpn_groups (id, name, description, prop_autologin, prop_deny, group_subnets, access_from, access_to, created_at) VALUES ($1,$2,$3,$4,FALSE,'[]'::jsonb,'[]'::jsonb,'[]'::jsonb,$5)`,
		id, input.Name, input.Description, input.PropAutologin, time.Now().UTC())
	if err != nil {
		return application.OpenVPNGroupView{}, mapPQLError(err, "vpn group")
	}
	return application.OpenVPNGroupView{ID: id, Name: input.Name, Description: input.Description, PropAutologin: input.PropAutologin}, nil
}

func (s *AdminPortalStore) UpdateOpenVPNGroup(ctx context.Context, groupname string, input application.OpenVPNGroupUpdateInput) (application.OpenVPNGroupView, error) {
	group, err := s.getOpenVPNGroup(ctx, groupname)
	if err != nil {
		return application.OpenVPNGroupView{}, err
	}
	if input.Description != nil {
		group.Description = strings.TrimSpace(*input.Description)
	}
	_, err = s.pool.Exec(ctx, `UPDATE admin_openvpn_groups SET description=$2 WHERE name=$1`, groupname, group.Description)
	if err != nil {
		return application.OpenVPNGroupView{}, err
	}
	return group, nil
}

func (s *AdminPortalStore) DeleteOpenVPNGroup(ctx context.Context, groupname string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM admin_openvpn_groups WHERE name = $1`, groupname)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "vpn group not found")
	}
	_, _ = s.pool.Exec(ctx, `UPDATE admin_openvpn_users SET group_name = '' WHERE group_name = $1`, groupname)
	return nil
}

func (s *AdminPortalStore) GetOpenVPNGroupProps(ctx context.Context, groupname string) (application.OpenVPNGroupProps, error) {
	group, err := s.getOpenVPNGroup(ctx, groupname)
	if err != nil {
		return application.OpenVPNGroupProps{}, err
	}
	return application.OpenVPNGroupProps{PropAutologin: group.PropAutologin, PropDeny: group.PropDeny, GroupSubnets: group.GroupSubnets, AccessFrom: group.AccessFrom, AccessTo: group.AccessTo}, nil
}

func (s *AdminPortalStore) SetOpenVPNGroupProps(ctx context.Context, groupname string, props application.OpenVPNGroupProps) error {
	tag, err := s.pool.Exec(ctx, `UPDATE admin_openvpn_groups SET prop_autologin=$2, prop_deny=$3, group_subnets=$4, access_from=$5, access_to=$6 WHERE name=$1`,
		groupname, props.PropAutologin, props.PropDeny, toJSON(props.GroupSubnets), toJSON(props.AccessFrom), toJSON(props.AccessTo))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "vpn group not found")
	}
	return nil
}

func (s *AdminPortalStore) GetOpenVPNGroupMembers(ctx context.Context, groupname string) ([]application.OpenVPNUserView, error) {
	all, err := s.ListOpenVPNUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]application.OpenVPNUserView, 0)
	for _, user := range all {
		if strings.EqualFold(user.Group, groupname) {
			out = append(out, user)
		}
	}
	return out, nil
}

func (s *AdminPortalStore) ListOpenVPNConnections(ctx context.Context, history bool) ([]application.OpenVPNConnectionView, error) {
	sql := `SELECT id, username, real_address, virtual_address, bytes_received, bytes_sent, connected_since, client_id, disconnected_at FROM admin_openvpn_connections WHERE active = TRUE ORDER BY connected_since DESC`
	if history {
		sql = `SELECT id, username, real_address, virtual_address, bytes_received, bytes_sent, connected_since, client_id, disconnected_at FROM admin_openvpn_connections ORDER BY connected_since DESC`
	}
	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.OpenVPNConnectionView, 0)
	for rows.Next() {
		var item application.OpenVPNConnectionView
		var connectedSince time.Time
		var disconnectedAt *time.Time
		if err := rows.Scan(&item.ID, &item.Username, &item.RealAddress, &item.VirtualAddress, &item.BytesReceived, &item.BytesSent, &connectedSince, &item.ClientID, &disconnectedAt); err != nil {
			return nil, err
		}
		item.ConnectedSince = connectedSince.Format(time.RFC3339)
		if disconnectedAt != nil {
			value := disconnectedAt.Format(time.RFC3339)
			item.DisconnectedAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AdminPortalStore) DisconnectOpenVPNConnection(ctx context.Context, clientID string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE admin_openvpn_connections SET active = FALSE, disconnected_at = NOW() WHERE client_id = $1 AND active = TRUE`, clientID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "connection not found")
	}
	return nil
}

func (s *AdminPortalStore) ListOpenVPNConfigs(ctx context.Context, username *string) ([]application.OpenVPNConfigView, error) {
	args := []any{}
	sql := `SELECT id, name, username, created_at, expires_at, download_count, status FROM admin_openvpn_configs`
	if username != nil && strings.TrimSpace(*username) != "" {
		sql += ` WHERE username = $1`
		args = append(args, strings.TrimSpace(*username))
	}
	sql += ` ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.OpenVPNConfigView, 0)
	for rows.Next() {
		var item application.OpenVPNConfigView
		var createdAt time.Time
		var expiresAt *time.Time
		if err := rows.Scan(&item.ID, &item.Name, &item.Username, &createdAt, &expiresAt, &item.DownloadCount, &item.Status); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		if expiresAt != nil {
			value := expiresAt.Format(time.RFC3339)
			item.ExpiresAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *AdminPortalStore) CreateOpenVPNConfig(ctx context.Context, input application.OpenVPNConfigCreateInput) (application.OpenVPNConfigView, error) {
	id := "ovcfg_" + strings.ToLower(strings.ReplaceAll(input.Name, " ", "_"))
	now := time.Now().UTC()
	qr := buildSVGDataURI(input.Name)
	_, err := s.pool.Exec(ctx, `INSERT INTO admin_openvpn_configs (id, name, username, platform, created_at, download_count, status, content, qr_code) VALUES ($1,$2,$3,'windows',$4,0,'active',$5,$6)`,
		id, input.Name, input.Username, now, sampleVPNConfig(input.Username), qr)
	if err != nil {
		return application.OpenVPNConfigView{}, mapPQLError(err, "vpn config")
	}
	return application.OpenVPNConfigView{ID: id, Name: input.Name, Username: input.Username, CreatedAt: now.Format(time.RFC3339), DownloadCount: 0, Status: "active"}, nil
}

func (s *AdminPortalStore) DownloadOpenVPNConfig(ctx context.Context, configID string) (application.OpenVPNConfigDownload, error) {
	var result application.OpenVPNConfigDownload
	err := s.pool.QueryRow(ctx, `SELECT name, content FROM admin_openvpn_configs WHERE id = $1`, configID).Scan(&result.Filename, &result.Content)
	if err != nil {
		if err == pgx.ErrNoRows {
			return application.OpenVPNConfigDownload{}, domainerr.New(domainerr.CodeNotFound, "vpn config not found")
		}
		return application.OpenVPNConfigDownload{}, err
	}
	result.Filename = result.Filename + ".ovpn"
	result.ContentType = "application/x-openvpn-profile"
	_, _ = s.pool.Exec(ctx, `UPDATE admin_openvpn_configs SET download_count = download_count + 1, last_downloaded_at = NOW() WHERE id = $1`, configID)
	return result, nil
}

func (s *AdminPortalStore) DeleteOpenVPNConfig(ctx context.Context, configID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM admin_openvpn_configs WHERE id = $1`, configID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "vpn config not found")
	}
	return nil
}

func (s *AdminPortalStore) RevokeOpenVPNConfig(ctx context.Context, configID string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE admin_openvpn_configs SET status = 'revoked' WHERE id = $1`, configID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "vpn config not found")
	}
	return nil
}

func (s *AdminPortalStore) GetOpenVPNConfigQRCode(ctx context.Context, configID string) (string, error) {
	var qrCode string
	err := s.pool.QueryRow(ctx, `SELECT qr_code FROM admin_openvpn_configs WHERE id = $1`, configID).Scan(&qrCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", domainerr.New(domainerr.CodeNotFound, "vpn config not found")
		}
		return "", err
	}
	return qrCode, nil
}

func (s *AdminPortalStore) getOpenVPNUser(ctx context.Context, username string) (application.OpenVPNUserView, error) {
	var user application.OpenVPNUserView
	var createdAt time.Time
	var lastLogin *time.Time
	var accessFromRaw, accessToRaw []byte
	err := s.pool.QueryRow(ctx, `SELECT id, username, email, enabled, created_at, last_login_at, status, assigned_ip, group_name, prop_autologin, prop_admin, prop_deny, prop_autogenerate, access_from, access_to FROM admin_openvpn_users WHERE username = $1`, username).
		Scan(&user.ID, &user.Username, &user.Email, &user.Enabled, &createdAt, &lastLogin, &user.Status, &user.AssignedIP, &user.Group, &user.PropAutologin, &user.PropAdmin, &user.PropDeny, &user.PropAutogenerate, &accessFromRaw, &accessToRaw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return application.OpenVPNUserView{}, domainerr.New(domainerr.CodeNotFound, "vpn user not found")
		}
		return application.OpenVPNUserView{}, err
	}
	user.CreatedAt = createdAt.Format(time.RFC3339)
	if lastLogin != nil {
		value := lastLogin.Format(time.RFC3339)
		user.LastLogin = &value
	}
	user.AccessFrom = decodeStringSlice(accessFromRaw)
	user.AccessTo = decodeStringSlice(accessToRaw)
	return user, nil
}

func (s *AdminPortalStore) getOpenVPNGroup(ctx context.Context, groupname string) (application.OpenVPNGroupView, error) {
	var group application.OpenVPNGroupView
	var groupSubnetsRaw, accessFromRaw, accessToRaw []byte
	err := s.pool.QueryRow(ctx, `SELECT id, name, description, prop_autologin, prop_deny, group_subnets, access_from, access_to FROM admin_openvpn_groups WHERE name = $1`, groupname).
		Scan(&group.ID, &group.Name, &group.Description, &group.PropAutologin, &group.PropDeny, &groupSubnetsRaw, &accessFromRaw, &accessToRaw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return application.OpenVPNGroupView{}, domainerr.New(domainerr.CodeNotFound, "vpn group not found")
		}
		return application.OpenVPNGroupView{}, err
	}
	group.GroupSubnets = decodeStringSlice(groupSubnetsRaw)
	group.AccessFrom = decodeStringSlice(accessFromRaw)
	group.AccessTo = decodeStringSlice(accessToRaw)
	return group, nil
}

func (s *AdminPortalStore) openVPNGroupCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `SELECT group_name, COUNT(*) FROM admin_openvpn_users WHERE group_name <> '' GROUP BY group_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		out[strings.ToLower(name)] = count
	}
	return out, rows.Err()
}

func applicationHashPassword(password string) string {
	return fmt.Sprintf("%x", sha256Sum(password))
}

func sha256Sum(value string) [32]byte {
	return sha256.Sum256([]byte(value))
}
