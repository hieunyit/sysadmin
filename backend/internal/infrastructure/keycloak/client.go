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
	if err := lockFileExclusive(lockFile); err != nil {
		_ = lockFile.Close()
		c.componentMu.Unlock()
		return nil, fmt.Errorf("lock component lock file failed: %w", err)
	}

	return func() {
		_ = unlockFile(lockFile)
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
		"requiredActions": func() []string {
			if req.RequiredActions == nil {
				return []string{}
			}
			return req.RequiredActions
		}(),
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

	return services.KeycloakUser{
		ID:              u.ID,
		Username:        u.Username,
		Email:           u.Email,
		DisplayName:     joinDisplayName(u.FirstName, u.LastName),
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		Enabled:         u.Enabled,
		EmailVerified:   u.EmailVerified,
		RequiredActions: u.RequiredActions,
		Attributes:      attributesFromKeycloak(u.Attributes),
		Groups:          groups,
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
	out := make([]services.KeycloakUser, 0, len(arr))
	for _, u := range arr {
		out = append(out, services.KeycloakUser{
			ID:              u.ID,
			Username:        u.Username,
			Email:           u.Email,
			DisplayName:     joinDisplayName(u.FirstName, u.LastName),
			FirstName:       u.FirstName,
			LastName:        u.LastName,
			Enabled:         u.Enabled,
			EmailVerified:   u.EmailVerified,
			RequiredActions: u.RequiredActions,
			Attributes:      attributesFromKeycloak(u.Attributes),
			Groups:          groupMap[u.ID],
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
