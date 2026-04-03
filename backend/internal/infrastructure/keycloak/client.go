package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	"backend/internal/config"
	"backend/internal/domain/services"
)

type Client struct {
	baseURL           string
	realm             string
	http              *http.Client
	timeout           time.Duration
	vpnEventClientID  string
	lookupConcurrency int
	ldapComponentID   string
	componentLockFile string
	componentMu       sync.Mutex
}

func New(cfg config.KeycloakConfig) *Client {
	oauthCfg := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
	}
	hc := oauthCfg.Client(context.Background())
	hc.Timeout = cfg.Timeout
	return &Client{
		baseURL:           strings.TrimRight(cfg.BaseURL, "/"),
		realm:             cfg.Realm,
		http:              hc,
		timeout:           cfg.Timeout,
		vpnEventClientID:  strings.TrimSpace(cfg.VPNEventClientID),
		lookupConcurrency: cfg.LookupConcurrency,
		ldapComponentID:   strings.TrimSpace(cfg.LDAPComponentID),
		componentLockFile: strings.TrimSpace(cfg.ComponentLockFile),
	}
}

type keycloakUserPayload struct {
	ID              string              `json:"id,omitempty"`
	Username        string              `json:"username"`
	Email           string              `json:"email"`
	Enabled         bool                `json:"enabled"`
	EmailVerified   bool                `json:"emailVerified"`
	FirstName       string              `json:"firstName,omitempty"`
	LastName        string              `json:"lastName,omitempty"`
	RequiredActions []string            `json:"requiredActions"`
	Attributes      map[string][]string `json:"attributes,omitempty"`
	Groups          []string            `json:"groups"`
}

type keycloakGroupPayload struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type keycloakUserGroupPayload struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type keycloakResetPasswordPayload struct {
	Temporary bool   `json:"temporary"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type keycloakComponentPayload struct {
	ID           string              `json:"id,omitempty"`
	Name         string              `json:"name,omitempty"`
	ProviderID   string              `json:"providerId,omitempty"`
	ProviderType string              `json:"providerType,omitempty"`
	ParentID     string              `json:"parentId,omitempty"`
	SubType      string              `json:"subType,omitempty"`
	Config       map[string][]string `json:"config,omitempty"`
}

type keycloakEventPayload struct {
	Time     int64             `json:"time"`
	ClientID string            `json:"clientId"`
	Details  map[string]string `json:"details"`
}

func (c *Client) adminURL(parts ...string) string {
	u, _ := url.Parse(c.baseURL)
	basePath := strings.Trim(u.Path, "/")
	p := []string{}
	if basePath != "" {
		p = append(p, basePath)
	}
	p = append(p, "admin", "realms", c.realm)
	p = append(p, parts...)
	u.Path = "/" + path.Join(p...)
	return u.String()
}

func splitDisplayName(display string) (string, string) {
	parts := strings.Fields(display)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[len(parts)-1], strings.Join(parts[:len(parts)-1], " ")
}

func joinDisplayName(first, last string) string {
	return strings.TrimSpace(last + " " + first)
}

func attributesToKeycloak(in map[string]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = []string{v}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func attributesFromKeycloak(in map[string][]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func (c *Client) effectiveLookupConcurrency() int {
	if c.lookupConcurrency > 0 {
		return c.lookupConcurrency
	}
	return 12
}

func extractIDFromLocation(location string) string {
	if strings.TrimSpace(location) == "" {
		return ""
	}
	parts := strings.Split(strings.TrimRight(location, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func normalizeGroupDisplay(g keycloakUserGroupPayload) string {
	if strings.TrimSpace(g.Path) != "" {
		return g.Path
	}
	return g.Name
}

func (c *Client) fetchUserGroups(ctx context.Context, userID string) ([]string, error) {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.adminURL("users", userID, "groups"), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get user groups failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var groups []keycloakUserGroupPayload
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		v := strings.TrimSpace(normalizeGroupDisplay(g))
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func (c *Client) getComponent(ctx context.Context, id string) (keycloakComponentPayload, error) {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.adminURL("components", id), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return keycloakComponentPayload{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return keycloakComponentPayload{}, fmt.Errorf("get component failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var component keycloakComponentPayload
	if err := json.NewDecoder(resp.Body).Decode(&component); err != nil {
		return keycloakComponentPayload{}, err
	}
	return component, nil
}

func (c *Client) updateComponent(ctx context.Context, component keycloakComponentPayload) error {
	body, _ := json.Marshal(component)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPut, c.adminURL("components", component.ID), bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update component failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return nil
}

func firstComponentConfigValue(config map[string][]string, key string) string {
	if len(config) == 0 {
		return ""
	}
	values := config[key]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func cloneComponentConfig(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for k, values := range in {
		if values == nil {
			out[k] = nil
			continue
		}
		cp := make([]string, len(values))
		copy(cp, values)
		out[k] = cp
	}
	return out
}

func (c *Client) backgroundContext() (context.Context, context.CancelFunc) {
	if c.timeout > 0 {
		return context.WithTimeout(context.Background(), c.timeout)
	}
	return context.Background(), func() {}
}

func (c *Client) lockComponentConfig() (func(), error) {
	c.componentMu.Lock()

	lockPath := strings.TrimSpace(c.componentLockFile)
	if lockPath == "" {
		return func() {
			c.componentMu.Unlock()
		}, nil
	}

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		c.componentMu.Unlock()
		return nil, fmt.Errorf("open component lock file failed: %w", err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		_ = lockFile.Close()
		c.componentMu.Unlock()
		return nil, fmt.Errorf("lock component lock file failed: %w", err)
	}

	return func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		_ = lockFile.Close()
		c.componentMu.Unlock()
	}, nil
}

func (c *Client) CreateUser(ctx context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
	identitySource := strings.ToLower(strings.TrimSpace(req.IdentitySource))
	switch identitySource {
	case "":
		return c.createUserRaw(ctx, req)
	case "local", "ldap":
		return c.createUserWithIdentitySource(ctx, req, identitySource)
	default:
		return services.KeycloakUser{}, fmt.Errorf("unsupported identity_source=%q", req.IdentitySource)
	}
}

func (c *Client) createUserRaw(ctx context.Context, req services.KeycloakUser) (services.KeycloakUser, error) {
	first := strings.TrimSpace(req.FirstName)
	last := strings.TrimSpace(req.LastName)
	if first == "" && last == "" && strings.TrimSpace(req.DisplayName) != "" {
		first, last = splitDisplayName(req.DisplayName)
	}
	payload := keycloakUserPayload{
		Username:        req.Username,
		Email:           req.Email,
		Enabled:         req.Enabled,
		EmailVerified:   req.EmailVerified,
		FirstName:       first,
		LastName:        last,
		RequiredActions: req.RequiredActions,
		Attributes:      attributesToKeycloak(req.Attributes),
		Groups:          req.Groups,
	}
	if payload.RequiredActions == nil {
		payload.RequiredActions = []string{}
	}
	if payload.Groups == nil {
		payload.Groups = []string{}
	}
	body, _ := json.Marshal(payload)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.adminURL("users"), bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return services.KeycloakUser{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return services.KeycloakUser{}, fmt.Errorf("create user failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	createdID := extractIDFromLocation(resp.Header.Get("Location"))
	if createdID == "" {
		// fallback for Keycloak versions not returning Location
		users, err := c.SearchUsers(ctx, req.Username, 0, 10)
		if err != nil {
			return services.KeycloakUser{}, err
		}
		for _, u := range users {
			if u.Username == req.Username {
				createdID = u.ID
				break
			}
		}
	}
	if createdID == "" {
		return services.KeycloakUser{}, fmt.Errorf("create user succeeded but could not resolve created user ID")
	}

	if strings.TrimSpace(req.Password) != "" {
		if err := c.ResetPassword(ctx, createdID, req.Password, req.PasswordTemporary); err != nil {
			rollbackCtx, cancel := c.backgroundContext()
			defer cancel()
			if rollbackErr := c.DeleteUser(rollbackCtx, createdID); rollbackErr != nil {
				return services.KeycloakUser{}, fmt.Errorf("create user succeeded but reset password failed and rollback delete failed: %w; rollback=%v", err, rollbackErr)
			}
			return services.KeycloakUser{}, fmt.Errorf("create user succeeded but reset password failed; rolled back created user %s: %w", createdID, err)
		}
	}

	return c.getUser(ctx, createdID, false)
}

func (c *Client) createUserWithIdentitySource(ctx context.Context, req services.KeycloakUser, identitySource string) (services.KeycloakUser, error) {
	if c.ldapComponentID == "" {
		return services.KeycloakUser{}, fmt.Errorf("identity_source=%s requires KEYCLOAK_LDAP_COMPONENT_ID to be configured", identitySource)
	}

	desiredSyncRegistrations := "true"
	if identitySource == "local" {
		desiredSyncRegistrations = "false"
	}

	releaseLock, err := c.lockComponentConfig()
	if err != nil {
		return services.KeycloakUser{}, err
	}
	defer releaseLock()

	component, err := c.getComponent(ctx, c.ldapComponentID)
	if err != nil {
		return services.KeycloakUser{}, fmt.Errorf("load LDAP component %s failed: %w", c.ldapComponentID, err)
	}

	originalConfig := cloneComponentConfig(component.Config)
	currentSyncRegistrations := firstComponentConfigValue(component.Config, "syncRegistrations")
	currentEditMode := firstComponentConfigValue(component.Config, "editMode")

	if identitySource == "ldap" && strings.EqualFold(strings.TrimSpace(currentEditMode), "READ_ONLY") {
		return services.KeycloakUser{}, fmt.Errorf("identity_source=ldap requires LDAP provider editMode to allow writes; current editMode=%q", currentEditMode)
	}

	changed := !strings.EqualFold(strings.TrimSpace(currentSyncRegistrations), desiredSyncRegistrations)
	if changed {
		if component.Config == nil {
			component.Config = map[string][]string{}
		}
		component.Config["syncRegistrations"] = []string{desiredSyncRegistrations}
		if err := c.updateComponent(ctx, component); err != nil {
			return services.KeycloakUser{}, fmt.Errorf("switch LDAP component syncRegistrations to %q failed: %w", desiredSyncRegistrations, err)
		}
	}

	user, createErr := c.createUserRaw(ctx, req)

	if changed {
		restoreCtx, cancel := c.backgroundContext()
		defer cancel()

		component.Config = cloneComponentConfig(originalConfig)
		if err := c.updateComponent(restoreCtx, component); err != nil {
			if createErr != nil {
				return services.KeycloakUser{}, fmt.Errorf("%w; additionally failed to restore LDAP component syncRegistrations to %q: %v", createErr, currentSyncRegistrations, err)
			}
			return services.KeycloakUser{}, fmt.Errorf("user may already have been created, but failed to restore LDAP component syncRegistrations to %q: %w", currentSyncRegistrations, err)
		}
	}

	if createErr != nil {
		return services.KeycloakUser{}, createErr
	}
	return user, nil
}

func (c *Client) UpdateUser(ctx context.Context, id string, req services.KeycloakUser) (services.KeycloakUser, error) {
	first := strings.TrimSpace(req.FirstName)
	last := strings.TrimSpace(req.LastName)
	if first == "" && last == "" && strings.TrimSpace(req.DisplayName) != "" {
		first, last = splitDisplayName(req.DisplayName)
	}
	payload := map[string]any{
		"id":            id,
		"username":      req.Username,
		"email":         req.Email,
		"enabled":       req.Enabled,
		"emailVerified": req.EmailVerified,
		"firstName":     first,
		"lastName":      last,
	}
	if attrs := attributesToKeycloak(req.Attributes); attrs != nil {
		payload["attributes"] = attrs
	}
	body, _ := json.Marshal(payload)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPut, c.adminURL("users", id), bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return services.KeycloakUser{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return services.KeycloakUser{}, fmt.Errorf("update user failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return c.getUser(ctx, id, true)
}

func (c *Client) DeleteUser(ctx context.Context, id string) error {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminURL("users", id), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete user failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) GetUser(ctx context.Context, id string) (services.KeycloakUser, error) {
	return c.getUser(ctx, id, true)
}

func (c *Client) getUser(ctx context.Context, id string, strictGroups bool) (services.KeycloakUser, error) {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.adminURL("users", id), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return services.KeycloakUser{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return services.KeycloakUser{}, fmt.Errorf("get user failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	var u keycloakUserPayload
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return services.KeycloakUser{}, err
	}

	groups, err := c.fetchUserGroups(ctx, id)
	if err != nil {
		if strictGroups {
			return services.KeycloakUser{}, fmt.Errorf("get user groups failed for %s: %w", id, err)
		}
		groups = nil
	}
	lastVPNLoginAt, err := c.fetchUserLastVPNLoginAt(ctx, id)
	lastVPNLoginLookupFailed := false
	if err != nil {
		lastVPNLoginAt = ""
		lastVPNLoginLookupFailed = true
	}

	return services.KeycloakUser{
		ID:                       u.ID,
		Username:                 u.Username,
		Email:                    u.Email,
		DisplayName:              joinDisplayName(u.FirstName, u.LastName),
		FirstName:                u.FirstName,
		LastName:                 u.LastName,
		Enabled:                  u.Enabled,
		EmailVerified:            u.EmailVerified,
		RequiredActions:          u.RequiredActions,
		Attributes:               attributesFromKeycloak(u.Attributes),
		Groups:                   groups,
		LastVPNLoginAt:           lastVPNLoginAt,
		LastVPNLoginLookupFailed: lastVPNLoginLookupFailed,
	}, nil
}

func (c *Client) SearchUsers(ctx context.Context, q string, first, max int) ([]services.KeycloakUser, error) {
	arr, err := c.searchUsersRaw(ctx, q, first, max)
	if err != nil {
		return nil, err
	}
	groupMap, err := c.fetchUserGroupsBatch(ctx, arr)
	if err != nil {
		return nil, err
	}
	lastVPNLoginMap, lastVPNLoginFailed := c.fetchUserLastVPNLoginBatch(ctx, arr)
	out := make([]services.KeycloakUser, 0, len(arr))
	for _, u := range arr {
		out = append(out, services.KeycloakUser{
			ID:                       u.ID,
			Username:                 u.Username,
			Email:                    u.Email,
			DisplayName:              joinDisplayName(u.FirstName, u.LastName),
			FirstName:                u.FirstName,
			LastName:                 u.LastName,
			Enabled:                  u.Enabled,
			EmailVerified:            u.EmailVerified,
			RequiredActions:          u.RequiredActions,
			Attributes:               attributesFromKeycloak(u.Attributes),
			Groups:                   groupMap[u.ID],
			LastVPNLoginAt:           lastVPNLoginMap[u.ID],
			LastVPNLoginLookupFailed: lastVPNLoginFailed[u.ID],
		})
	}
	return out, nil
}

func (c *Client) searchUsersRaw(ctx context.Context, q string, first, max int) ([]keycloakUserPayload, error) {
	u, _ := url.Parse(c.adminURL("users"))
	query := u.Query()
	if q != "" {
		query.Set("search", q)
	}
	query.Set("first", fmt.Sprintf("%d", first))
	query.Set("max", fmt.Sprintf("%d", max))
	u.RawQuery = query.Encode()
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search users failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	var arr []keycloakUserPayload
	if err := json.NewDecoder(resp.Body).Decode(&arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func (c *Client) LookupUsers(ctx context.Context, usernames []string) (map[string]services.KeycloakUserLookup, error) {
	targets := normalizeLookupTargets(usernames)
	if len(targets) == 0 {
		return map[string]services.KeycloakUserLookup{}, nil
	}

	var (
		rawUsers []keycloakUserPayload
		err      error
	)
	if len(targets) <= c.effectiveLookupConcurrency()*2 {
		rawUsers, err = c.lookupUsersByExactSearch(ctx, targets)
	} else {
		rawUsers, err = c.lookupUsersByPagedScan(ctx, targets)
	}
	if err != nil {
		return nil, err
	}

	lastVPNLoginMap, lastVPNLoginFailed := c.fetchUserLastVPNLoginBatch(ctx, rawUsers)

	out := make(map[string]services.KeycloakUserLookup, len(rawUsers))
	for _, user := range rawUsers {
		username := strings.ToLower(strings.TrimSpace(user.Username))
		if username == "" {
			continue
		}
		expiry := ""
		if user.Attributes != nil {
			expiry = strings.TrimSpace(attributesFromKeycloak(user.Attributes)["userExpiryVPN"])
		}
		out[username] = services.KeycloakUserLookup{
			Username:                 strings.TrimSpace(user.Username),
			Email:                    strings.TrimSpace(user.Email),
			DisplayName:              strings.TrimSpace(joinDisplayName(user.FirstName, user.LastName)),
			Enabled:                  user.Enabled,
			VPNExpireAt:              expiry,
			LastVPNLoginAt:           strings.TrimSpace(lastVPNLoginMap[user.ID]),
			LastVPNLoginLookupFailed: lastVPNLoginFailed[user.ID],
		}
	}
	return out, nil
}

func (c *Client) fetchUserLastVPNLoginAt(ctx context.Context, userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", nil
	}

	u, _ := url.Parse(c.adminURL("events"))
	query := u.Query()
	query.Set("user", userID)
	query.Set("first", "0")
	query.Set("max", "101")
	if targetClientID := strings.TrimSpace(c.vpnEventClientID); targetClientID != "" {
		query.Set("client", targetClientID)
	}
	u.RawQuery = query.Encode()

	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get user events failed: status=%d body=%s", resp.StatusCode, string(b))
	}

	var events []keycloakEventPayload
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return "", err
	}

	return selectLatestVPNEventTime(events, c.vpnEventClientID), nil
}

func normalizeLookupTargets(usernames []string) []string {
	if len(usernames) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(usernames))
	out := make([]string, 0, len(usernames))
	for _, username := range usernames {
		normalized := strings.ToLower(strings.TrimSpace(username))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func (c *Client) lookupUsersByExactSearch(ctx context.Context, targets []string) ([]keycloakUserPayload, error) {
	results := make(map[string]keycloakUserPayload, len(targets))
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		errOnce  sync.Once
	)

	sem := make(chan struct{}, c.effectiveLookupConcurrency())
	for _, username := range targets {
		username := username
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			page, err := c.searchUsersRaw(ctx, username, 0, 20)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
				})
				return
			}
			for _, user := range page {
				if !strings.EqualFold(strings.TrimSpace(user.Username), username) {
					continue
				}
				mu.Lock()
				results[username] = user
				mu.Unlock()
				return
			}
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	out := make([]keycloakUserPayload, 0, len(results))
	for _, username := range targets {
		if user, ok := results[username]; ok {
			out = append(out, user)
		}
	}
	return out, nil
}

func (c *Client) lookupUsersByPagedScan(ctx context.Context, targets []string) ([]keycloakUserPayload, error) {
	remaining := make(map[string]struct{}, len(targets))
	for _, username := range targets {
		remaining[username] = struct{}{}
	}

	const pageSize = 200
	first := 0
	results := make(map[string]keycloakUserPayload, len(targets))
	for len(remaining) > 0 {
		page, err := c.searchUsersRaw(ctx, "", first, pageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		for _, user := range page {
			username := strings.ToLower(strings.TrimSpace(user.Username))
			if username == "" {
				continue
			}
			if _, ok := remaining[username]; !ok {
				continue
			}
			results[username] = user
			delete(remaining, username)
		}
		if len(page) < pageSize {
			break
		}
		first += pageSize
	}

	out := make([]keycloakUserPayload, 0, len(results))
	for _, username := range targets {
		if user, ok := results[username]; ok {
			out = append(out, user)
		}
	}
	return out, nil
}

func (c *Client) fetchUserLastVPNLoginBatch(ctx context.Context, users []keycloakUserPayload) (map[string]string, map[string]bool) {
	if len(users) == 0 {
		return map[string]string{}, map[string]bool{}
	}

	results := make(map[string]string, len(users))
	failed := make(map[string]bool, len(users))
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, c.effectiveLookupConcurrency())
	for _, user := range users {
		userID := strings.TrimSpace(user.ID)
		if userID == "" {
			continue
		}

		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			lastLoginAt, err := c.fetchUserLastVPNLoginAt(ctx, id)
			if err != nil {
				mu.Lock()
				results[id] = ""
				failed[id] = true
				mu.Unlock()
				return
			}

			mu.Lock()
			results[id] = lastLoginAt
			mu.Unlock()
		}(userID)
	}

	wg.Wait()
	return results, failed
}

func selectLatestVPNEventTime(events []keycloakEventPayload, targetClientID string) string {
	var latest int64
	targetClientID = strings.TrimSpace(targetClientID)
	targetHost := ""
	if targetClientID != "" {
		if parsed, err := url.Parse(targetClientID); err == nil {
			targetHost = strings.ToLower(strings.TrimSpace(parsed.Host))
		}
	}

	for _, event := range events {
		if !matchesVPNEvent(event, targetClientID, targetHost) {
			continue
		}
		if event.Time > latest {
			latest = event.Time
		}
	}
	if latest <= 0 {
		return ""
	}
	loc := time.FixedZone("UTC+7", 7*60*60)
	return time.UnixMilli(latest).In(loc).Format(time.RFC3339)
}

func matchesVPNEvent(event keycloakEventPayload, targetClientID, targetHost string) bool {
	clientID := strings.TrimSpace(event.ClientID)
	if targetClientID == "" {
		return clientID != ""
	}
	if strings.EqualFold(clientID, targetClientID) {
		return true
	}
	if targetHost == "" || len(event.Details) == 0 {
		return false
	}
	redirectURI := strings.TrimSpace(event.Details["redirect_uri"])
	if redirectURI == "" {
		return false
	}
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(parsed.Host), targetHost)
}

func (c *Client) fetchUserGroupsBatch(ctx context.Context, users []keycloakUserPayload) (map[string][]string, error) {
	if len(users) == 0 {
		return map[string][]string{}, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(map[string][]string, len(users))
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
		errOnce  sync.Once
	)

	sem := make(chan struct{}, c.effectiveLookupConcurrency())
	for _, user := range users {
		userID := user.ID
		if strings.TrimSpace(userID) == "" {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			groups, err := c.fetchUserGroups(ctx, userID)
			if err != nil {
				errOnce.Do(func() {
					firstErr = fmt.Errorf("search users fetch groups failed for %s: %w", userID, err)
					cancel()
				})
				return
			}

			mu.Lock()
			results[userID] = groups
			mu.Unlock()
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (c *Client) ResetPassword(ctx context.Context, userID, password string, temporary bool) error {
	payload := keycloakResetPasswordPayload{
		Temporary: temporary,
		Type:      "password",
		Value:     password,
	}
	body, _ := json.Marshal(payload)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPut, c.adminURL("users", userID, "reset-password"), bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reset password failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) CreateGroup(ctx context.Context, req services.KeycloakGroup) (services.KeycloakGroup, error) {
	payload := keycloakGroupPayload{Name: req.Name}
	body, _ := json.Marshal(payload)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.adminURL("groups"), bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return services.KeycloakGroup{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return services.KeycloakGroup{}, fmt.Errorf("create group failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	groups, err := c.SearchGroups(ctx, req.Name)
	if err != nil {
		return services.KeycloakGroup{}, err
	}
	for _, g := range groups {
		if g.Name == req.Name {
			return g, nil
		}
	}
	return services.KeycloakGroup{}, fmt.Errorf("create group succeeded but unable to resolve group")
}

func (c *Client) UpdateGroup(ctx context.Context, id string, req services.KeycloakGroup) (services.KeycloakGroup, error) {
	payload := keycloakGroupPayload{ID: id, Name: req.Name}
	body, _ := json.Marshal(payload)
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPut, c.adminURL("groups", id), bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(hreq)
	if err != nil {
		return services.KeycloakGroup{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return services.KeycloakGroup{}, fmt.Errorf("update group failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return c.GetGroup(ctx, id)
}

func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminURL("groups", id), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete group failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) GetGroup(ctx context.Context, id string) (services.KeycloakGroup, error) {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.adminURL("groups", id), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return services.KeycloakGroup{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return services.KeycloakGroup{}, fmt.Errorf("get group failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	var g keycloakGroupPayload
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return services.KeycloakGroup{}, err
	}
	return services.KeycloakGroup{ID: g.ID, Name: g.Name, Path: g.Path}, nil
}

func (c *Client) SearchGroups(ctx context.Context, q string) ([]services.KeycloakGroup, error) {
	u, _ := url.Parse(c.adminURL("groups"))
	params := u.Query()
	if q != "" {
		params.Set("search", q)
	}
	u.RawQuery = params.Encode()
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search groups failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	var arr []keycloakGroupPayload
	if err := json.NewDecoder(resp.Body).Decode(&arr); err != nil {
		return nil, err
	}
	out := make([]services.KeycloakGroup, 0, len(arr))
	for _, g := range arr {
		out = append(out, services.KeycloakGroup{ID: g.ID, Name: g.Name, Path: g.Path})
	}
	return out, nil
}

func (c *Client) ListGroupMembers(ctx context.Context, groupID string) ([]services.KeycloakUser, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, fmt.Errorf("group id is required")
	}

	const pageSize = 200
	first := 0
	out := make([]services.KeycloakUser, 0)
	for {
		u, _ := url.Parse(c.adminURL("groups", groupID, "members"))
		query := u.Query()
		query.Set("first", fmt.Sprintf("%d", first))
		query.Set("max", fmt.Sprintf("%d", pageSize))
		u.RawQuery = query.Encode()

		hreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		resp, err := c.http.Do(hreq)
		if err != nil {
			return nil, err
		}

		var page []keycloakUserPayload
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("list group members failed: status=%d body=%s", resp.StatusCode, string(b))
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, user := range page {
			out = append(out, services.KeycloakUser{
				ID:          user.ID,
				Username:    strings.TrimSpace(user.Username),
				Email:       strings.TrimSpace(user.Email),
				DisplayName: strings.TrimSpace(joinDisplayName(user.FirstName, user.LastName)),
				FirstName:   strings.TrimSpace(user.FirstName),
				LastName:    strings.TrimSpace(user.LastName),
				Enabled:     user.Enabled,
				Attributes:  attributesFromKeycloak(user.Attributes),
			})
		}
		if len(page) < pageSize {
			break
		}
		first += pageSize
	}
	return out, nil
}

func (c *Client) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodPut, c.adminURL("users", userID, "groups", groupID), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add user to group failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *Client) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	hreq, _ := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminURL("users", userID, "groups", groupID), nil)
	resp, err := c.http.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove user from group failed: status=%d body=%s", resp.StatusCode, string(b))
	}
	return nil
}
