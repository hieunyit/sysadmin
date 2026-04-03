package openvpn

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"

	"backend/internal/config"
	"backend/internal/domain/services"
)

const authHeader = "X-OpenVPN-As-AuthToken"

type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client

	mu    sync.RWMutex
	token string
}

func New(cfg config.OpenVPNConfig) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &Client{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		username: cfg.Username,
		password: cfg.Password,
		http:     &http.Client{Timeout: cfg.Timeout, Transport: transport},
	}
}

type authResponse struct {
	AuthToken string `json:"auth_token"`
}

func (c *Client) login(ctx context.Context) error {
	payload := map[string]any{
		"username":      c.username,
		"password":      c.password,
		"request_admin": true,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/login/userpassword", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openvpn login failed status=%d body=%s", resp.StatusCode, string(body))
	}
	var out authResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if out.AuthToken == "" {
		return fmt.Errorf("openvpn login response missing auth_token")
	}
	c.mu.Lock()
	c.token = out.AuthToken
	c.mu.Unlock()
	return nil
}

func (c *Client) getToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, out any) (*http.Response, error) {
	if c.getToken() == "" {
		if err := c.login(ctx); err != nil {
			return nil, err
		}
	}
	try := 0
	for {
		try++
		var body io.Reader
		if payload != nil {
			b, _ := json.Marshal(payload)
			body = bytes.NewReader(b)
		}
		req, _ := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(authHeader, c.getToken())

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized && try == 1 {
			_ = resp.Body.Close()
			if err := c.login(ctx); err != nil {
				return nil, err
			}
			continue
		}

		if out != nil {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				b, _ := io.ReadAll(resp.Body)
				return resp, fmt.Errorf("openvpn request %s %s failed status=%d body=%s", method, path, resp.StatusCode, string(b))
			}
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
				return resp, err
			}
			return resp, nil
		}

		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			return resp, fmt.Errorf("openvpn request %s %s failed status=%d body=%s", method, path, resp.StatusCode, string(b))
		}
		return resp, nil
	}
}

func (c *Client) AddRuleset(ctx context.Context, name, comment string) (int64, error) {
	payload := map[string]any{"name": name, "comment": comment}
	var out struct {
		ID int64 `json:"id"`
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/access/rulesets/add", payload, &out)
	return out.ID, err
}

func (c *Client) UpdateRuleset(ctx context.Context, id int64, name, comment string) error {
	payload := map[string]any{"id": id, "name": name, "comment": comment}
	_, err := c.doJSON(ctx, http.MethodPost, "/access/rulesets/update", payload, nil)
	return err
}

func (c *Client) DeleteRulesets(ctx context.Context, ids []int64) error {
	payload := map[string]any{"ids": ids}
	_, err := c.doJSON(ctx, http.MethodPost, "/access/rulesets/delete", payload, nil)
	return err
}

func (c *Client) ListRulesets(ctx context.Context, owner, nameFilter string) ([]services.OpenVPNRuleset, error) {
	payload := map[string]any{"owner": owner}
	if nameFilter != "" {
		payload["filters"] = map[string]any{"name": map[string]any{"operation": "substring", "value": nameFilter}}
	}
	var out struct {
		Rulesets []struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			Comment   string `json:"comment"`
			Owner     string `json:"owner"`
			OwnerType string `json:"owner_type"`
			Position  int    `json:"position"`
		} `json:"rulesets"`
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/access/rulesets/list", payload, &out)
	if err != nil {
		return nil, err
	}
	res := make([]services.OpenVPNRuleset, 0, len(out.Rulesets))
	for _, rs := range out.Rulesets {
		res = append(res, services.OpenVPNRuleset{
			ID:        rs.ID,
			Name:      rs.Name,
			Comment:   rs.Comment,
			Owner:     rs.Owner,
			OwnerType: rs.OwnerType,
			Position:  rs.Position,
		})
	}
	return res, nil
}

func (c *Client) ModifyRules(ctx context.Context, addOrUpdate []services.OpenVPNRule, deleteIDs []int64) ([]int64, error) {
	add := make([]map[string]any, 0, len(addOrUpdate))
	for _, r := range addOrUpdate {
		item := map[string]any{
			"ruleset_id": r.RulesetID,
			"type":       r.Type,
			"match_type": r.MatchType,
			"match_data": r.MatchData,
			"action":     r.Action,
			"position":   r.Position,
			"comment":    r.Comment,
		}
		if r.ID != nil {
			item["id"] = *r.ID
		}
		add = append(add, item)
	}
	payload := map[string]any{}
	if len(add) > 0 {
		payload["add"] = add
	}
	if len(deleteIDs) > 0 {
		payload["delete"] = deleteIDs
	}
	if len(payload) == 0 {
		return []int64{}, nil
	}
	var out struct {
		Added []int64 `json:"added"`
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/access/rules/modify", payload, &out)
	if err != nil {
		return nil, err
	}
	return out.Added, nil
}

func (c *Client) ModifyUserRulesetMapping(ctx context.Context, add map[string][]services.SubjectRulesetRef, del map[string][]int64) error {
	addPayload := map[string][]map[string]any{}
	for username, refs := range add {
		if len(refs) == 0 {
			continue
		}
		arr := make([]map[string]any, 0, len(refs))
		for _, ref := range refs {
			arr = append(arr, map[string]any{"ruleset_id": ref.RulesetID, "position": ref.Position})
		}
		addPayload[username] = arr
	}
	delPayload := map[string]map[string]any{}
	for username, ids := range del {
		if len(ids) == 0 {
			continue
		}
		delPayload[username] = map[string]any{"ruleset_ids": ids}
	}
	payload := map[string]any{}
	if len(addPayload) > 0 {
		payload["add"] = addPayload
	}
	if len(delPayload) > 0 {
		payload["delete"] = delPayload
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/access/user-rulesets/modify", payload, nil)
	return err
}

func makeAccessRouteObject(item services.AccessRouteItem) (map[string]any, error) {
	obj := map[string]any{
		"type":   item.RouteType,
		"accept": item.Accept,
	}
	if item.RouteType == "user" && item.Username != nil {
		obj["username"] = *item.Username
	}
	if item.RouteType == "group" && item.Groupname != nil {
		obj["groupname"] = *item.Groupname
	}
	if item.CIDR != nil && (item.RouteType == "route" || item.RouteType == "nat") {
		pfx, err := netip.ParsePrefix(*item.CIDR)
		if err != nil {
			return nil, err
		}
		subnet := map[string]any{
			"ipv6":          pfx.Addr().Is6(),
			"netip":         pfx.Addr().String(),
			"prefix_length": pfx.Bits(),
		}
		if item.ServiceSpec != nil && strings.TrimSpace(*item.ServiceSpec) != "" && !strings.EqualFold(strings.TrimSpace(*item.ServiceSpec), "all") {
			services, err := parseAccessServiceSpec(*item.ServiceSpec)
			if err != nil {
				return nil, err
			}
			if len(services) > 0 {
				subnet["service"] = services
			}
		}
		obj["subnet"] = subnet
	}
	return obj, nil
}

func parseAccessServiceSpec(spec string) ([]map[string]any, error) {
	parts := strings.Split(spec, ",")
	out := make([]map[string]any, 0, len(parts))
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" || strings.EqualFold(part, "all") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(part), "icmp") {
			out = append(out, map[string]any{
				"protocol": "icmp",
				"type":     part,
			})
			continue
		}

		sep := ":"
		if strings.Contains(part, "/") {
			sep = "/"
		}
		tokens := strings.SplitN(part, sep, 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid port spec %q: use tcp:443, udp:53 or tcp:443-445", part)
		}

		protocol := strings.ToLower(strings.TrimSpace(tokens[0]))
		if protocol != "tcp" && protocol != "udp" {
			return nil, fmt.Errorf("unsupported protocol %q in port spec %q", protocol, part)
		}

		rangeSpec := strings.TrimSpace(tokens[1])
		if rangeSpec == "" {
			return nil, fmt.Errorf("missing port in spec %q", part)
		}

		item := map[string]any{"protocol": protocol}
		if strings.Contains(rangeSpec, "-") {
			bounds := strings.SplitN(rangeSpec, "-", 2)
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid port range %q", part)
			}
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port in %q: %w", part, err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port in %q: %w", part, err)
			}
			if err := validatePortNumber(start); err != nil {
				return nil, fmt.Errorf("invalid start port in %q: %w", part, err)
			}
			if err := validatePortNumber(end); err != nil {
				return nil, fmt.Errorf("invalid end port in %q: %w", part, err)
			}
			if end < start {
				return nil, fmt.Errorf("invalid port range %q: end port must be greater than or equal to start port", part)
			}
			item["start_port"] = start
			item["end_port"] = end
		} else {
			start, err := strconv.Atoi(rangeSpec)
			if err != nil {
				return nil, fmt.Errorf("invalid port in %q: %w", part, err)
			}
			if err := validatePortNumber(start); err != nil {
				return nil, fmt.Errorf("invalid port in %q: %w", part, err)
			}
			item["start_port"] = start
		}
		out = append(out, item)
	}
	return out, nil
}

func validatePortNumber(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func makeAccessRouteItems(items []services.AccessRouteItem) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := map[string]any{"type": item.Type}
		if item.Username != nil {
			row["username"] = *item.Username
		}
		if item.Groupname != nil {
			row["groupname"] = *item.Groupname
		}
		route, err := makeAccessRouteObject(item)
		if err != nil {
			return nil, err
		}
		row["access_route"] = route
		out = append(out, row)
	}
	return out, nil
}

func (c *Client) SetAccessList(ctx context.Context, items []services.AccessRouteItem) error {
	rows, err := makeAccessRouteItems(items)
	if err != nil {
		return err
	}
	payload := map[string]any{"items_set": rows}
	_, err = c.doJSON(ctx, http.MethodPost, "/userprop/access/set", payload, nil)
	return err
}

func (c *Client) AppendAccessList(ctx context.Context, items []services.AccessRouteItem) error {
	rows, err := makeAccessRouteItems(items)
	if err != nil {
		return err
	}
	payload := map[string]any{"items_append": rows}
	_, err = c.doJSON(ctx, http.MethodPost, "/userprop/access/set", payload, nil)
	return err
}

func (c *Client) RemoveAccessList(ctx context.Context, items []services.AccessRouteItem) error {
	rows, err := makeAccessRouteItems(items)
	if err != nil {
		return err
	}
	payload := map[string]any{"items_remove": rows}
	_, err = c.doJSON(ctx, http.MethodPost, "/userprop/access/set", payload, nil)
	return err
}

func (c *Client) ListUsers(ctx context.Context, q services.OpenVPNUserListQuery) (map[string]any, error) {
	pageSize := q.Limit
	if pageSize <= 0 {
		pageSize = 50
	}
	payload := map[string]any{
		"page_size": pageSize,
	}
	if q.Offset > 0 {
		payload["offset"] = q.Offset
	}
	if strings.TrimSpace(q.Search) != "" {
		payload["filters"] = map[string]any{
			"name": map[string]any{
				"operation": "substring",
				"value":     q.Search,
			},
		}
	}
	var out map[string]any
	_, err := c.doJSON(ctx, http.MethodPost, "/users/list", payload, &out)
	return out, err
}

func (c *Client) ListGroups(ctx context.Context, q services.OpenVPNGroupListQuery) (map[string]any, error) {
	pageSize := q.Limit
	if pageSize <= 0 {
		pageSize = 50
	}
	payload := map[string]any{
		"page_size":         pageSize,
		"enumerate_members": q.EnumerateMembers,
	}
	if q.Offset > 0 {
		payload["offset"] = q.Offset
	}
	if strings.TrimSpace(q.Search) != "" {
		payload["filters"] = map[string]any{
			"name": map[string]any{
				"operation": "substring",
				"value":     q.Search,
			},
		}
	}
	var out map[string]any
	_, err := c.doJSON(ctx, http.MethodPost, "/groups/list", payload, &out)
	return out, err
}

func (c *Client) ListAccessLists(ctx context.Context, q services.OpenVPNAccessListQuery) (map[string]any, error) {
	payload := map[string]any{}
	username := strings.TrimSpace(q.Username)
	groupname := strings.TrimSpace(q.Groupname)
	subjectType := strings.TrimSpace(q.SubjectType)

	if username != "" {
		payload["users"] = []string{username}
	}
	if groupname != "" {
		payload["groups"] = []string{groupname}
	}
	// Keep payload minimal and aligned with AS API behavior:
	// - when users/groups are provided, AS may still return mixed rows.
	// - avoid over-constraining with object_type filter at request time.
	if username == "" && groupname == "" && subjectType != "" {
		payload["filters"] = map[string]any{
			"object_type": subjectType,
		}
	}
	var out map[string]any
	if _, err := c.doJSON(ctx, http.MethodPost, "/userprop/access/list", payload, &out); err != nil {
		return nil, err
	}
	filterAccessListProfiles(out, username, groupname, subjectType)
	return out, nil
}

func filterAccessListProfiles(out map[string]any, username, groupname, subjectType string) {
	raw, ok := out["profiles"]
	if !ok {
		return
	}
	arr, ok := raw.([]any)
	if !ok {
		return
	}

	keep := make([]any, 0, len(arr))
	for _, item := range arr {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rowUser := strings.TrimSpace(formatMapString(row, "username"))
		rowGroup := strings.TrimSpace(formatMapString(row, "groupname"))

		if username != "" && !strings.EqualFold(rowUser, username) {
			continue
		}
		if groupname != "" && !strings.EqualFold(rowGroup, groupname) {
			continue
		}
		if subjectType == "user" && rowUser == "" {
			continue
		}
		if subjectType == "group" && rowGroup == "" {
			continue
		}
		keep = append(keep, row)
	}
	out["profiles"] = keep
	if _, ok := out["total"]; ok {
		out["total"] = len(keep)
	}
}

func formatMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (c *Client) ListRules(ctx context.Context, q services.OpenVPNRuleListQuery) (map[string]any, error) {
	payload := map[string]any{
		"ruleset_ids": q.RulesetIDs,
	}
	filters := map[string]any{}
	if strings.TrimSpace(q.Type) != "" {
		filters["type"] = map[string]any{
			"operation": "equal",
			"value":     q.Type,
		}
	}
	if strings.TrimSpace(q.MatchType) != "" {
		filters["match_type"] = map[string]any{
			"operation": "equal",
			"value":     q.MatchType,
		}
	}
	if strings.TrimSpace(q.MatchData) != "" {
		filters["match_data"] = map[string]any{
			"operation": "substring",
			"value":     q.MatchData,
		}
	}
	if len(filters) > 0 {
		payload["filters"] = filters
	}
	var out map[string]any
	_, err := c.doJSON(ctx, http.MethodPost, "/access/rules/list", payload, &out)
	return out, err
}

func (c *Client) EnsureUser(ctx context.Context, username string) error {
	payload := map[string]any{
		"name":        username,
		"auth_method": "saml",
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/users/create", payload, nil)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func (c *Client) EnsureGroup(ctx context.Context, groupname string) error {
	payload := map[string]any{
		"name":        groupname,
		"auth_method": "saml",
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/groups/create", payload, nil)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func (c *Client) SetUserGroup(ctx context.Context, username, groupname string) error {
	payload := []map[string]any{
		{
			"name":  username,
			"group": groupname,
		},
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/userprop/set", payload, nil)
	return err
}

func (c *Client) DeleteUser(ctx context.Context, username string) error {
	payload := map[string]any{"users": []string{username}}
	resp, err := c.doJSON(ctx, http.MethodPost, "/users/delete", payload, nil)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return err
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return nil
}

func (c *Client) DeleteGroup(ctx context.Context, groupname string) error {
	payload := map[string]any{"groups": []string{groupname}}
	resp, err := c.doJSON(ctx, http.MethodPost, "/groups/delete", payload, nil)
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return nil
}

func parseRuleID(in string) (*int64, error) {
	if in == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

var _ services.OpenVPNPolicyService = (*Client)(nil)
