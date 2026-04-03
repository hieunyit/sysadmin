package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"backend/cmd/sysctl/internal"
	"backend/pkg/inputvalidate"
)

type rootOptions struct {
	Server string
	Token  string
	Output string
}

func main() {
	opts := &rootOptions{}
	root := &cobra.Command{
		Use:           "sysctl",
		Short:         "CLI for Keycloak/SSO and OpenVPN operations",
		Long:          "Thin CLI for automation. Business logic lives in the backend REST API.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `sysctl keycloak user list --output table
sysctl keycloak group list --output table
sysctl openvpn group access-list list --groupname vanhanh --output table`,
	}
	root.PersistentFlags().StringVar(
		&opts.Server,
		"server",
		getenv("SYSCTL_SERVER", "http://127.0.0.1:8080"),
		"API server base URL",
	)
	root.PersistentFlags().StringVar(
		&opts.Token,
		"token",
		getenv("SYSCTL_TOKEN", ""),
		"Admin API token (or set SYSCTL_TOKEN)",
	)
	root.PersistentFlags().StringVar(&opts.Output, "output", "json", "Output format: json|table")

	root.AddCommand(keycloakCmd(opts))
	root.AddCommand(openvpnCmd(opts))

	root.CompletionOptions.HiddenDefaultCmd = true

	if err := root.Execute(); err != nil {
		if shouldShowHelp(err) {
			helpCmd := nearestCommand(root, os.Args[1:])
			_ = helpCmd.Help()
			fmt.Fprintln(os.Stderr)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func api(opts *rootOptions) *internal.Client {
	return internal.New(opts.Server, opts.Token)
}

func printOutput(outFormat string, raw []byte) {
	printWarnings(extractWarnings(raw))
	if outFormat == "table" {
		var data any
		if err := json.Unmarshal(raw, &data); err != nil {
			fmt.Println(string(raw))
			return
		}
		printTable(data)
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		fmt.Println(string(raw))
		return
	}
	fmt.Println(pretty.String())
}

func extractWarnings(raw []byte) []string {
	var envelope struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	return dedupeWarnings(nil, envelope.Warnings...)
}

func printWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "Warnings:")
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		fmt.Fprintln(os.Stderr, "- "+warning)
	}
	fmt.Fprintln(os.Stderr)
}

func printTable(v any) {
	root, ok := v.(map[string]any)
	if !ok {
		fmt.Printf("%v\n", v)
		return
	}
	data := root["data"]
	if m, ok := data.(map[string]any); ok {
		switch {
		case m["items"] != nil:
			data = m["items"]
		case m["profiles"] != nil:
			data = m["profiles"]
		case m["rules"] != nil:
			data = m["rules"]
		}
	}
	data = normalizeTableData(data)

	rows, cols := tableRowsAndCols(data)
	if len(cols) == 0 {
		fmt.Println("| value |")
		fmt.Println("| --- |")
		for _, r := range rows {
			fmt.Printf("| %s |\n", r["value"])
		}
		return
	}
	printPipeTable(rows, cols)
}

func normalizeTableData(data any) any {
	arr, ok := data.([]any)
	if !ok || len(arr) == 0 {
		return data
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return data
	}
	if looksLikeOpenVPNUserProfile(first) {
		return normalizeOpenVPNUserProfiles(arr)
	}
	if _, ok := first["access_route"]; !ok {
		return data
	}
	out := make([]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		row := map[string]any{}
		username := formatCell(m["username"])
		groupname := formatCell(m["groupname"])
		row["username"] = username
		row["groupname"] = groupname
		switch {
		case username != "":
			row["subject_type"] = "user"
			row["subject"] = username
		case groupname != "":
			row["subject_type"] = "group"
			row["subject"] = groupname
		}
		row["access_type"] = m["type"]

		if route, ok := m["access_route"].(map[string]any); ok {
			row["route_type"] = route["type"]
			row["accept"] = route["accept"]
			if subnet, ok := route["subnet"].(map[string]any); ok {
				row["netip"] = subnet["netip"]
				row["prefix_length"] = subnet["prefix_length"]
				row["ipv6"] = subnet["ipv6"]
				if services, ok := subnet["service"].([]any); ok {
					row["service"] = flattenServices(services)
				}
			}
		}

		out = append(out, row)
	}
	return out
}

func looksLikeOpenVPNUserProfile(row map[string]any) bool {
	if row == nil {
		return false
	}
	if _, ok := row["name"]; !ok {
		return false
	}
	if _, ok := row["password_defined"]; ok {
		return true
	}
	if _, ok := row["mfa_status"]; ok {
		return true
	}
	if _, ok := row["static_ipv4"]; ok {
		return true
	}
	return false
}

func normalizeOpenVPNUserProfiles(items []any) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, map[string]any{
			"username":         formatCell(row["name"]),
			"group":            formatCell(row["group"]),
			"auth_method":      propertyValue(row["auth_method"]),
			"deny":             propertyValue(row["deny"]),
			"password_defined": formatCell(row["password_defined"]),
			"mfa_status":       formatCell(row["mfa_status"]),
		})
	}
	return out
}

func propertyValue(v any) string {
	if v == nil {
		return ""
	}
	if m, ok := v.(map[string]any); ok {
		if value, exists := m["value"]; exists {
			return formatCell(value)
		}
	}
	return formatCell(v)
}

func flattenServices(items []any) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		protocol := formatCell(m["protocol"])
		start := formatCell(m["start_port"])
		end := formatCell(m["end_port"])
		switch {
		case protocol != "" && start != "" && end != "":
			parts = append(parts, fmt.Sprintf("%s:%s-%s", protocol, start, end))
		case protocol != "" && start != "":
			parts = append(parts, fmt.Sprintf("%s:%s", protocol, start))
		case protocol != "":
			parts = append(parts, protocol)
		}
	}
	return strings.Join(parts, ",")
}

func tableRowsAndCols(data any) ([]map[string]string, []string) {
	var rows []map[string]string
	colSet := map[string]struct{}{}

	addRow := func(m map[string]any) {
		row := map[string]string{}
		for k, v := range m {
			if strings.HasSuffix(k, "_lookup_failed") {
				continue
			}
			row[k] = formatTableCell(k, v)
			colSet[k] = struct{}{}
		}
		rows = append(rows, row)
	}

	switch t := data.(type) {
	case []any:
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				addRow(m)
				continue
			}
			row := map[string]string{"value": formatCell(item)}
			rows = append(rows, row)
			colSet["value"] = struct{}{}
		}
	case map[string]any:
		addRow(t)
	default:
		row := map[string]string{"value": formatCell(t)}
		rows = append(rows, row)
		colSet["value"] = struct{}{}
	}

	cols := make([]string, 0, len(colSet))
	for c := range colSet {
		cols = append(cols, c)
	}
	sort.Strings(cols)
	return rows, orderTableColumns(cols)
}

func formatTableCell(column string, v any) string {
	return formatCell(v)
}

func orderTableColumns(cols []string) []string {
	preferred := []string{
		"id",
		"username",
		"group",
		"email",
		"display_name",
		"auth_method",
		"deny",
		"password_defined",
		"mfa_status",
		"enabled",
		"vpn_enabled",
		"vpn_access_state",
		"groups",
		"subject_type",
		"subject",
		"kind",
		"entry_type",
		"target",
		"action",
		"accept",
		"port",
		"comment",
	}
	rank := make(map[string]int, len(preferred))
	for i, col := range preferred {
		rank[col] = i
	}
	sort.SliceStable(cols, func(i, j int) bool {
		ri, okI := rank[cols[i]]
		rj, okJ := rank[cols[j]]
		switch {
		case okI && okJ:
			return ri < rj
		case okI:
			return true
		case okJ:
			return false
		default:
			return cols[i] < cols[j]
		}
	})
	return cols
}

func formatCell(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// JSON numbers are float64 by default
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

func dedupeWarnings(current []string, warnings ...string) []string {
	seen := make(map[string]struct{}, len(current))
	for _, warning := range current {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		seen[warning] = struct{}{}
	}
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		if _, ok := seen[warning]; ok {
			continue
		}
		seen[warning] = struct{}{}
		current = append(current, warning)
	}
	return current
}

func printPipeTable(rows []map[string]string, cols []string) {
	widths := map[string]int{}
	for _, c := range cols {
		widths[c] = len(c)
	}
	for _, r := range rows {
		for _, c := range cols {
			if l := len(r[c]); l > widths[c] {
				widths[c] = l
			}
		}
	}

	// Header
	fmt.Print("|")
	for _, c := range cols {
		fmt.Printf(" %-*s |", widths[c], c)
	}
	fmt.Println()

	// Separator
	fmt.Print("|")
	for _, c := range cols {
		fmt.Printf("-%s-|", strings.Repeat("-", widths[c]))
	}
	fmt.Println()

	// Rows
	for _, r := range rows {
		fmt.Print("|")
		for _, c := range cols {
			fmt.Printf(" %-*s |", widths[c], r[c])
		}
		fmt.Println()
	}
}

func printFlatMap(v any) {
	m, ok := v.(map[string]any)
	if !ok {
		fmt.Printf("%v\n", v)
		return
	}
	for k, val := range m {
		fmt.Printf("%-20s %v\n", k, val)
	}
}

func shouldShowHelp(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "required flag") ||
		strings.Contains(msg, "accepts ") ||
		strings.Contains(msg, "requires at least")
}

func nearestCommand(root *cobra.Command, args []string) *cobra.Command {
	current := root
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		next := findSubcommand(current, arg)
		if next == nil {
			break
		}
		current = next
	}
	return current
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
		for _, alias := range sub.Aliases {
			if alias == name {
				return sub
			}
		}
	}
	return nil
}

func resolveAccessListSubject(cmd *cobra.Command, subjectType string) (string, error) {
	flagName := "username"
	if subjectType == "group" {
		flagName = "groupname"
	}
	subject, _ := cmd.Flags().GetString(flagName)
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", fmt.Errorf("--%s is required", flagName)
	}
	return subject, nil
}

func buildAccessListBodyFromFlags(cmd *cobra.Command, subjectType, subject string) (any, error) {
	file, _ := cmd.Flags().GetString("file")
	if strings.TrimSpace(file) != "" {
		payload, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		body := map[string]any{}
		if err := json.Unmarshal(payload, &body); err != nil {
			return nil, err
		}
		if err := applyAccessListOwnerToPayload(body, subjectType, subject); err != nil {
			return nil, err
		}
		return body, nil
	}

	target, _ := cmd.Flags().GetString("target")
	port, _ := cmd.Flags().GetString("port")

	subject = strings.TrimSpace(subject)
	target = strings.TrimSpace(target)
	port = strings.TrimSpace(port)

	if subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if target == "" {
		return nil, fmt.Errorf("--target is required when --file is not used")
	}
	if port != "" && !strings.EqualFold(port, "all") && !looksLikeCIDROrIP(target) {
		return nil, fmt.Errorf("--port only applies to IP/CIDR targets")
	}

	item := map[string]any{
		"target": target,
	}
	switch subjectType {
	case "user":
		item["username"] = subject
	case "group":
		item["groupname"] = subject
	default:
		return nil, fmt.Errorf("unsupported access-list subject type: %s", subjectType)
	}
	if port != "" && !strings.EqualFold(port, "all") {
		item["service_spec"] = port
	}

	return map[string]any{"items": []map[string]any{item}}, nil
}

func applyAccessListOwnerToPayload(body map[string]any, subjectType, subject string) error {
	rawItems, ok := body["items"]
	if !ok {
		return fmt.Errorf("payload file must include items")
	}
	items, ok := rawItems.([]any)
	if !ok {
		return fmt.Errorf("payload file field items must be an array")
	}
	if len(items) == 0 {
		return fmt.Errorf("payload file field items must not be empty")
	}

	for idx, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("payload file items[%d] must be objects", idx)
		}
		username := strings.TrimSpace(formatCell(item["username"]))
		groupname := strings.TrimSpace(formatCell(item["groupname"]))
		switch subjectType {
		case "user":
			if groupname != "" {
				return fmt.Errorf("items[%d].groupname is not allowed for user access-list command", idx)
			}
			if username != "" && !strings.EqualFold(username, subject) {
				return fmt.Errorf("items[%d].username must match --username", idx)
			}
			item["username"] = subject
			delete(item, "groupname")
		case "group":
			if username != "" {
				return fmt.Errorf("items[%d].username is not allowed for group access-list command", idx)
			}
			if groupname != "" && !strings.EqualFold(groupname, subject) {
				return fmt.Errorf("items[%d].groupname must match --groupname", idx)
			}
			item["groupname"] = subject
			delete(item, "username")
		default:
			return fmt.Errorf("unsupported access-list subject type: %s", subjectType)
		}
	}

	body["items"] = items
	return nil
}

func looksLikeCIDROrIP(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	if _, err := netip.ParsePrefix(input); err == nil {
		return true
	}
	if _, err := netip.ParseAddr(input); err == nil {
		return true
	}
	return false
}

type userListItem struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Enabled     bool     `json:"enabled"`
	Groups      []string `json:"groups"`
}

type userListResponse struct {
	Data []userListItem `json:"data"`
}

type userReference struct {
	ID       string
	Username string
	Email    string
}

type groupListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type groupListResponse struct {
	Data []groupListItem `json:"data"`
}

func buildUserListPath(q string, limit, offset int) string {
	params := url.Values{}
	params.Set("q", q)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("offset", fmt.Sprintf("%d", offset))
	return "/api/v1/keycloak/users?" + params.Encode()
}

func fetchUsers(opts *rootOptions, q string, limit, offset int) ([]byte, userListResponse, error) {
	path := buildUserListPath(q, limit, offset)
	body, _, err := api(opts).Do(context.Background(), "GET", path, nil)
	if err != nil {
		return nil, userListResponse{}, err
	}
	var out userListResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, userListResponse{}, err
	}
	return body, out, nil
}

func buildGroupListPath(q string) string {
	params := url.Values{}
	params.Set("q", q)
	return "/api/v1/keycloak/groups?" + params.Encode()
}

func fetchGroups(opts *rootOptions, q string) ([]byte, groupListResponse, error) {
	path := buildGroupListPath(q)
	body, _, err := api(opts).Do(context.Background(), "GET", path, nil)
	if err != nil {
		return nil, groupListResponse{}, err
	}
	var out groupListResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, groupListResponse{}, err
	}
	return body, out, nil
}

func resolveUserReference(id, username, email string) (userReference, error) {
	ref := userReference{
		ID:       strings.TrimSpace(id),
		Username: strings.TrimSpace(username),
		Email:    strings.TrimSpace(email),
	}
	count := 0
	if ref.ID != "" {
		count++
	}
	if ref.Username != "" {
		count++
	}
	if ref.Email != "" {
		count++
	}
	switch count {
	case 0:
		return userReference{}, fmt.Errorf("set one of --id, --username or --email")
	case 1:
		return ref, nil
	default:
		return userReference{}, fmt.Errorf("set only one of --id, --username or --email")
	}
}

func resolveUserID(opts *rootOptions, id, username, email string) (string, error) {
	ref, err := resolveUserReference(id, username, email)
	if err != nil {
		return "", err
	}
	if ref.ID != "" {
		return ref.ID, nil
	}

	query := ref.Username
	matchLabel := "username"
	matchValue := ref.Username
	if ref.Email != "" {
		query = ref.Email
		matchLabel = "email"
		matchValue = ref.Email
	}

	limit := 100
	offset := 0
	seen := map[string]struct{}{}
	matches := make([]userListItem, 0, 1)
	for {
		_, resp, err := fetchUsers(opts, query, limit, offset)
		if err != nil {
			return "", err
		}
		if len(resp.Data) == 0 {
			break
		}
		for _, item := range resp.Data {
			if item.ID == "" {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}

			switch {
			case ref.Username != "" && strings.EqualFold(strings.TrimSpace(item.Username), ref.Username):
				matches = append(matches, item)
			case ref.Email != "" && strings.EqualFold(strings.TrimSpace(item.Email), ref.Email):
				matches = append(matches, item)
			}
		}
		if len(resp.Data) < limit {
			break
		}
		offset += limit
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%s %q not found", matchLabel, matchValue)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("%s %q resolved to multiple users", matchLabel, matchValue)
	}
}

func resolveGroupID(opts *rootOptions, id, name string) (string, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)

	switch {
	case id != "" && name != "":
		return "", fmt.Errorf("set either --id or --name, not both")
	case id != "":
		return id, nil
	case name == "":
		return "", fmt.Errorf("set either --id or --name")
	}

	_, resp, err := fetchGroups(opts, name)
	if err != nil {
		return "", err
	}

	matches := make([]groupListItem, 0, 1)
	seen := map[string]struct{}{}
	for _, item := range resp.Data {
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}

		if strings.EqualFold(strings.TrimSpace(item.Name), name) || strings.EqualFold(strings.TrimSpace(item.Path), name) {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("group %q not found", name)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("group %q resolved to multiple groups; use --id or full path", name)
	}
}

func parseStrictBoolFlag(raw, flagName string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("--%s must be true or false", flagName)
	}
}

func validateIdentitySourceCreateFlags(identitySource, userType, companyName, onboardDate, phone, department, manager string) error {
	identitySource = strings.ToLower(strings.TrimSpace(identitySource))
	userType = strings.ToLower(strings.TrimSpace(userType))
	onboardDate = strings.TrimSpace(onboardDate)

	switch identitySource {
	case "local":
		missing := make([]string, 0, 2)
		if userType == "" {
			missing = append(missing, "--user-type")
		}
		if strings.TrimSpace(companyName) == "" {
			missing = append(missing, "--company-name")
		}
		if len(missing) > 0 {
			return fmt.Errorf("create-local requires %s", strings.Join(missing, ", "))
		}
		if userType != "partner" && userType != "outsource" {
			return fmt.Errorf("--user-type must be partner or outsource for local users")
		}
	case "ldap":
		missing := make([]string, 0, 4)
		if userType == "" {
			missing = append(missing, "--user-type")
		}
		if strings.TrimSpace(phone) == "" {
			missing = append(missing, "--phone")
		}
		if strings.TrimSpace(department) == "" {
			missing = append(missing, "--department")
		}
		if strings.TrimSpace(manager) == "" {
			missing = append(missing, "--manager")
		}
		if len(missing) > 0 {
			return fmt.Errorf("create-ldap requires %s", strings.Join(missing, ", "))
		}
		if userType != "employee" {
			return fmt.Errorf("--user-type must be employee for ldap users")
		}
	}
	if onboardDate != "" {
		if _, err := inputvalidate.NormalizeDDMMYYYY(onboardDate); err != nil {
			return fmt.Errorf("--onboard-date must be a valid date in dd/MM/yyyy")
		}
	}

	return nil
}

func buildUserCreatePayload(cmd *cobra.Command, forcedIdentitySource string) (map[string]any, error) {
	username, _ := cmd.Flags().GetString("username")
	email, _ := cmd.Flags().GetString("email")
	notificationEmail, _ := cmd.Flags().GetString("notification-email")
	firstName, _ := cmd.Flags().GetString("first-name")
	lastName, _ := cmd.Flags().GetString("last-name")
	displayName, _ := cmd.Flags().GetString("display-name")
	identitySource, _ := cmd.Flags().GetString("identity-source")
	enabled, _ := cmd.Flags().GetBool("enabled")
	emailVerified, _ := cmd.Flags().GetBool("email-verified")
	password, _ := cmd.Flags().GetString("password")
	passwordTemporary, _ := cmd.Flags().GetBool("password-temporary")
	fullName, _ := cmd.Flags().GetString("full-name")
	userType, _ := cmd.Flags().GetString("user-type")
	companyName, _ := cmd.Flags().GetString("company-name")
	groups, _ := cmd.Flags().GetStringSlice("groups")
	phone, _ := cmd.Flags().GetString("phone")
	department, _ := cmd.Flags().GetString("department")
	manager, _ := cmd.Flags().GetString("manager")
	employeeID, _ := cmd.Flags().GetString("employee-id")
	onboardDate, _ := cmd.Flags().GetString("onboard-date")
	workAddress, _ := cmd.Flags().GetString("work-address")
	state, _ := cmd.Flags().GetString("state")

	if forcedIdentitySource != "" {
		identitySource = forcedIdentitySource
	}
	identitySource = strings.ToLower(strings.TrimSpace(identitySource))
	userType = strings.ToLower(strings.TrimSpace(userType))
	if err := validateIdentitySourceCreateFlags(identitySource, userType, companyName, onboardDate, phone, department, manager); err != nil {
		return nil, err
	}
	if identitySource == "ldap" || identitySource == "local" {
		passwordTemporary = true
	}
	if onboardDate != "" {
		normalizedOnboardDate, _ := inputvalidate.NormalizeDDMMYYYY(onboardDate)
		onboardDate = normalizedOnboardDate
	}

	attributes := map[string]any{
		"fullName":    fullName,
		"phone":       phone,
		"department":  department,
		"manager":     manager,
		"employeeID":  employeeID,
		"onboardDate": onboardDate,
		"workAddress": strings.TrimSpace(workAddress),
		"State":       state,
		"userType":    userType,
		"companyName": companyName,
	}

	return map[string]any{
		"username":           username,
		"email":              email,
		"notification_email": strings.TrimSpace(notificationEmail),
		"first_name":         firstName,
		"last_name":          lastName,
		"display_name":       displayName,
		"identity_source":    identitySource,
		"enabled":            enabled,
		"email_verified":     emailVerified,
		"password":           password,
		"password_temporary": passwordTemporary,
		"required_actions":   []string{},
		"groups":             groups,
		"attributes":         attributes,
	}, nil
}

func configureUserCreateFlags(create *cobra.Command, includeIdentityFlag bool) {
	create.Flags().String("username", "", "Username")
	create.Flags().String("email", "", "Email")
	create.Flags().String("notification-email", "", "Optional email to receive account-created notification instead of the account email")
	create.Flags().String("first-name", "", "First name")
	create.Flags().String("last-name", "", "Last name")
	create.Flags().String("display-name", "", "Display name (optional)")
	if includeIdentityFlag {
		create.Flags().String("identity-source", "", "User source: local|ldap. Requires KEYCLOAK_LDAP_COMPONENT_ID when set")
	}
	create.Flags().String("password", "", "Initial password; if omitted, the backend generates a random temporary password")
	create.Flags().Bool("password-temporary", false, "Force user đổi password khi đăng nhập lần đầu")
	create.Flags().Bool("enabled", true, "Enable user")
	create.Flags().Bool("email-verified", true, "Set emailVerified=true/false")
	create.Flags().String("full-name", "", "attributes.fullName")
	create.Flags().String("phone", "", "attributes.phone")
	create.Flags().String("department", "", "attributes.department")
	create.Flags().String("manager", "", "attributes.manager")
	create.Flags().String("employee-id", "", "attributes.employeeID")
	create.Flags().String("onboard-date", "", "attributes.onboardDate (dd/MM/yyyy) for LDAP onboarding email")
	create.Flags().String("work-address", "", "attributes.workAddress for LDAP onboarding email")
	create.Flags().String("state", "", "attributes.State")
	create.Flags().String("user-type", "", "attributes.userType (employee|partner|outsource)")
	create.Flags().String("company-name", "", "attributes.companyName")
	create.Flags().StringSlice("groups", []string{}, "Keycloak group paths or IDs, separated by commas")
	_ = create.MarkFlagRequired("username")
	_ = create.MarkFlagRequired("email")
	_ = create.MarkFlagRequired("first-name")
	_ = create.MarkFlagRequired("last-name")
	_ = create.MarkFlagRequired("full-name")
}

func newUserCreateCommand(opts *rootOptions, use, short, longDesc, example, forcedIdentitySource string) *cobra.Command {
	create := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    longDesc,
		Example: example,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, err := buildUserCreatePayload(cmd, forcedIdentitySource)
			if err != nil {
				return err
			}
			b, _, err := api(opts).Do(context.Background(), "POST", "/api/v1/keycloak/users", payload)
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	configureUserCreateFlags(create, forcedIdentitySource == "")
	return create
}

func normalizeCSVHeader(header string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(header)))
}

func csvValue(row map[string]string, keys ...string) string {
	for _, key := range keys {
		if v, ok := row[normalizeCSVHeader(key)]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func csvBool(row map[string]string, defaultValue bool, keys ...string) (bool, error) {
	raw := csvValue(row, keys...)
	if raw == "" {
		return defaultValue, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", raw)
	}
}

func csvList(row map[string]string, keys ...string) []string {
	raw := csvValue(row, keys...)
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '|' || r == ';'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func loadUserImportRows(filePath string, delimiter rune) ([]map[string]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	if delimiter != 0 {
		reader.Comma = delimiter
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("csv file is empty")
	}

	headers := make([]string, len(records[0]))
	for i, header := range records[0] {
		headers[i] = normalizeCSVHeader(header)
	}

	rows := make([]map[string]string, 0, len(records)-1)
	for idx, record := range records[1:] {
		if len(record) == 0 {
			continue
		}
		row := make(map[string]string, len(headers))
		nonEmpty := false
		for col, header := range headers {
			value := ""
			if col < len(record) {
				value = strings.TrimSpace(record[col])
			}
			if value != "" {
				nonEmpty = true
			}
			row[header] = value
		}
		if !nonEmpty {
			continue
		}
		row[normalizeCSVHeader("_row_number")] = fmt.Sprintf("%d", idx+2)
		rows = append(rows, row)
	}
	return rows, nil
}

func buildUserImportPayload(row map[string]string, defaultIdentitySource string) (map[string]any, error) {
	enabled, err := csvBool(row, true, "enabled")
	if err != nil {
		return nil, err
	}
	emailVerified, err := csvBool(row, true, "email_verified", "emailVerified")
	if err != nil {
		return nil, err
	}
	passwordTemporary, err := csvBool(row, false, "password_temporary", "passwordTemporary")
	if err != nil {
		return nil, err
	}
	identitySource := csvValue(row, "identity_source", "identitySource")
	if identitySource == "" {
		identitySource = strings.TrimSpace(defaultIdentitySource)
	}
	if identitySource == "ldap" || identitySource == "local" {
		passwordTemporary = true
	}
	if err := validateIdentitySourceCreateFlags(
		identitySource,
		csvValue(row, "userType", "user_type"),
		csvValue(row, "companyName", "company_name"),
		csvValue(row, "onboardDate", "onboard_date", "onboard"),
		csvValue(row, "phone"),
		csvValue(row, "department"),
		csvValue(row, "manager"),
	); err != nil {
		return nil, fmt.Errorf("row %s: %v", csvValue(row, "_row_number"), err)
	}

	attributes := map[string]any{
		"fullName":    csvValue(row, "fullName", "full_name"),
		"phone":       csvValue(row, "phone"),
		"department":  csvValue(row, "department"),
		"manager":     csvValue(row, "manager"),
		"employeeID":  csvValue(row, "employeeID", "employee_id"),
		"onboardDate": csvValue(row, "onboardDate", "onboard_date", "onboard"),
		"workAddress": csvValue(row, "workAddress", "work_address", "address"),
		"State":       csvValue(row, "State", "state"),
		"userType":    csvValue(row, "userType", "user_type"),
		"companyName": csvValue(row, "companyName", "company_name"),
	}
	if onboardDate := csvValue(row, "onboardDate", "onboard_date", "onboard"); onboardDate != "" {
		normalizedOnboardDate, err := inputvalidate.NormalizeDDMMYYYY(onboardDate)
		if err != nil {
			return nil, fmt.Errorf("row %s: onboardDate must be a valid date in dd/MM/yyyy", csvValue(row, "_row_number"))
		}
		attributes["onboardDate"] = normalizedOnboardDate
	}

	return map[string]any{
		"username":           csvValue(row, "username"),
		"email":              csvValue(row, "email"),
		"notification_email": csvValue(row, "notification_email", "notificationEmail", "notify_email", "notifyEmail"),
		"first_name":         csvValue(row, "first_name", "firstName"),
		"last_name":          csvValue(row, "last_name", "lastName"),
		"display_name":       csvValue(row, "display_name", "displayName"),
		"identity_source":    identitySource,
		"enabled":            enabled,
		"email_verified":     emailVerified,
		"password":           csvValue(row, "password"),
		"password_temporary": passwordTemporary,
		"required_actions":   csvList(row, "required_actions", "requiredActions"),
		"groups":             csvList(row, "groups"),
		"attributes":         attributes,
	}, nil
}

func userImportTemplateHeaders() []string {
	return []string{
		"username",
		"email",
		"notification_email",
		"first_name",
		"last_name",
		"display_name",
		"password",
		"password_temporary",
		"enabled",
		"email_verified",
		"identity_source",
		"groups",
		"required_actions",
		"fullName",
		"phone",
		"department",
		"manager",
		"employeeID",
		"onboardDate",
		"workAddress",
		"State",
		"userType",
		"companyName",
	}
}

func userImportTemplateRows(preset string) ([][]string, error) {
	localRow := []string{
		"test.local1",
		"test.local1@mbfs.vn",
		"",
		"Local",
		"One",
		"Local One",
		"",
		"true",
		"true",
		"true",
		"local",
		"",
		"",
		"Local One",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"partner",
		"TEST",
	}
	ldapRow := []string{
		"test.ldap1",
		"test.ldap1@mbfs.vn",
		"nguoi.nhan.thongbao@mbfs.vn",
		"Ldap",
		"One",
		"Ldap One",
		"",
		"true",
		"true",
		"true",
		"ldap",
		"",
		"",
		"Ldap One",
		"0987654321",
		"Phong Van hanh",
		"CN=Nguyen Dinh Truong,OU=MBFS_Users,DC=mbfs,DC=local",
		"12345",
		"15/04/2026",
		"Tòa nhà MobiFone, Hà Nội",
		"",
		"employee",
		"",
	}

	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "", "mixed":
		return [][]string{localRow, ldapRow}, nil
	case "local":
		return [][]string{localRow}, nil
	case "ldap":
		return [][]string{ldapRow}, nil
	default:
		return nil, fmt.Errorf("--preset must be one of: mixed, local, ldap")
	}
}

func writeUserImportTemplate(filePath string, delimiter rune, preset string, headerOnly, force bool) (int, error) {
	if strings.TrimSpace(filePath) == "" {
		return 0, fmt.Errorf("--file is required")
	}
	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return 0, fmt.Errorf("file %q already exists; rerun with --force to overwrite", filePath)
		} else if !os.IsNotExist(err) {
			return 0, err
		}
	}

	f, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	if delimiter != 0 {
		writer.Comma = delimiter
	}
	if err := writer.Write(userImportTemplateHeaders()); err != nil {
		return 0, err
	}

	rowCount := 0
	if !headerOnly {
		rows, err := userImportTemplateRows(preset)
		if err != nil {
			return 0, err
		}
		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				return rowCount, err
			}
			rowCount++
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return rowCount, err
	}
	return rowCount, nil
}

func writeOpenVPNUserExportCSV(filePath string, rows []map[string]any) error {
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("--file is required")
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	headers := []string{
		"username",
		"group",
		"auth_method",
		"deny",
		"password_defined",
		"mfa_status",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		record := make([]string, 0, len(headers))
		for _, header := range headers {
			record = append(record, formatCell(row[header]))
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func userCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "user", Short: "Manage users"}

	create := newUserCreateCommand(
		opts,
		"create",
		"Create user",
		"Create a Keycloak user. Leave --identity-source empty to use the current realm behavior. Set --identity-source local|ldap to ask the backend to switch the configured LDAP provider syncRegistrations flag before user creation and restore it afterward. If --password is omitted, the backend generates a random temporary password automatically. Use --notification-email if the account-created notification should go to a different mailbox than the account email.",
		`sysctl keycloak user create --username test.hieuny --email test.hieuny@mbfs.vn --first-name "Hiếu" --last-name "Nguyễn Y" --full-name "Nguyễn Y Hiếu" --enabled --email-verified
sysctl keycloak user create --username local.user --email local.user@example.com --first-name "Local" --last-name "User" --full-name "Local User" --password 'Mbfs@111' --identity-source local`,
		"",
	)

	createLocal := newUserCreateCommand(
		opts,
		"create-local",
		"Create local Keycloak user",
		"Create a local Keycloak user. The backend temporarily switches the configured LDAP provider syncRegistrations=false for this request, creates the user, then restores the previous provider config. Local users require --full-name, --user-type partner|outsource and --company-name. password_temporary is forced to true. If --password is omitted, the backend generates a random temporary password automatically. Use --notification-email if the account-created notification should go to a different mailbox than the account email.",
		`sysctl keycloak user create-local --username test.local --email test.local@mbfs.vn --first-name "Local" --last-name "User" --full-name "Local User" --user-type partner --company-name "TEST"`,
		"local",
	)

	createLDAP := newUserCreateCommand(
		opts,
		"create-ldap",
		"Create LDAP-backed user",
		"Create an LDAP-backed user. The backend temporarily switches the configured LDAP provider syncRegistrations=true for this request, creates the user, then restores the previous provider config. LDAP users require --full-name, --phone, --department, --manager and --user-type employee. password_temporary is forced to true. If --password is omitted, the backend generates a random temporary password automatically. The LDAP provider editMode must already allow writes. Use --onboard-date and --work-address to fill the LDAP onboarding email. Use --notification-email to send the account-created email to a different mailbox instead of the LDAP account email.",
		`sysctl keycloak user create-ldap --username test.ldap --email test.ldap@mbfs.vn --notification-email nguoi.nhan@example.com --first-name "Ldap" --last-name "User" --full-name "Ldap User" --phone 0987654321 --department "Trung tâm Dịch vụ Chuyển đổi số" --manager "CN=Nguyễn Đình Trường,OU=MBFS_Users,DC=mbfs,DC=local" --user-type employee --onboard-date "15/04/2026" --work-address "Tòa nhà MobiFone, Hà Nội"`,
		"ldap",
	)

	importCSV := &cobra.Command{
		Use:   "import-csv",
		Short: "Import users from CSV",
		Long:  "Import users from a CSV file by calling the same backend user-create API for each row. Supported headers include username,email,notification_email,first_name,last_name,display_name,password,password_temporary,enabled,email_verified,identity_source,groups,required_actions,fullName,phone,department,manager,employeeID,onboardDate,workAddress,State,userType,companyName. For multi-value columns such as groups and required_actions, separate values with | or ; inside the CSV cell. LDAP rows require userType=employee and phone/department/manager. Local rows require userType=partner|outsource and companyName. onboardDate, when set, must use dd/MM/yyyy. For both local and ldap rows, password_temporary is forced to true. If the password column is left empty, the backend generates a random temporary password. If notification_email is set, the account-created email goes there instead of the account email.",
		Example: `sysctl keycloak user import-csv --file examples/users.import.csv
sysctl keycloak user import-csv --file users.csv
sysctl keycloak user import-csv --file ldap-users.csv --identity-source ldap
sysctl keycloak user import-csv --file local-users.csv --identity-source local --delimiter ';'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			delimiter, _ := cmd.Flags().GetString("delimiter")
			defaultIdentitySource, _ := cmd.Flags().GetString("identity-source")
			stopOnError, _ := cmd.Flags().GetBool("stop-on-error")

			if defaultIdentitySource != "" && defaultIdentitySource != "local" && defaultIdentitySource != "ldap" {
				return fmt.Errorf("--identity-source must be local or ldap")
			}

			var comma rune = ','
			if delimiter != "" {
				runes := []rune(delimiter)
				if len(runes) != 1 {
					return fmt.Errorf("--delimiter must be a single character")
				}
				comma = runes[0]
			}

			rows, err := loadUserImportRows(filePath, comma)
			if err != nil {
				return err
			}

			results := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				payload, err := buildUserImportPayload(row, defaultIdentitySource)
				rowNumber := csvValue(row, "_row_number")
				username := csvValue(row, "username")
				email := csvValue(row, "email")
				identitySource := csvValue(row, "identity_source", "identitySource")
				if identitySource == "" {
					identitySource = strings.TrimSpace(defaultIdentitySource)
				}
				if err != nil {
					results = append(results, map[string]any{
						"row":             rowNumber,
						"username":        username,
						"email":           email,
						"identity_source": identitySource,
						"status":          "failed",
						"message":         err.Error(),
					})
					if stopOnError {
						break
					}
					continue
				}

				body, _, err := api(opts).Do(context.Background(), "POST", "/api/v1/keycloak/users", payload)
				if err != nil {
					results = append(results, map[string]any{
						"row":             rowNumber,
						"username":        username,
						"email":           email,
						"identity_source": identitySource,
						"status":          "failed",
						"message":         err.Error(),
					})
					if stopOnError {
						break
					}
					continue
				}

				var resp struct {
					Data map[string]any `json:"data"`
				}
				_ = json.Unmarshal(body, &resp)
				results = append(results, map[string]any{
					"row":             rowNumber,
					"username":        username,
					"email":           email,
					"identity_source": identitySource,
					"status":          "created",
					"id":              formatCell(resp.Data["id"]),
				})
			}

			output, _ := json.Marshal(map[string]any{"data": results})
			printOutput(opts.Output, output)
			return nil
		},
	}
	importCSV.Flags().String("file", "", "CSV file path")
	importCSV.Flags().String("delimiter", ",", "CSV delimiter, for example , or ;")
	importCSV.Flags().String("identity-source", "", "Default identity source for all rows: local|ldap")
	importCSV.Flags().Bool("stop-on-error", false, "Stop import immediately when a row fails")
	_ = importCSV.MarkFlagRequired("file")

	_ = createLocal.MarkFlagRequired("user-type")
	_ = createLocal.MarkFlagRequired("company-name")

	_ = createLDAP.MarkFlagRequired("user-type")
	_ = createLDAP.MarkFlagRequired("phone")
	_ = createLDAP.MarkFlagRequired("department")
	_ = createLDAP.MarkFlagRequired("manager")

	importTemplate := &cobra.Command{
		Use:   "import-template",
		Short: "Generate CSV template for user import",
		Long:  "Generate a CSV template that matches the user import format. Use --preset mixed for both local and ldap sample rows, or --preset local|ldap for a single-mode template. Use --header-only if you only want the CSV header row.",
		Example: `sysctl keycloak user import-template --file users.template.csv
sysctl keycloak user import-template --file users.local.csv --preset local
sysctl keycloak user import-template --file users.ldap.csv --preset ldap
sysctl keycloak user import-template --file users.header.csv --header-only --force`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			filePath, _ := cmd.Flags().GetString("file")
			preset, _ := cmd.Flags().GetString("preset")
			headerOnly, _ := cmd.Flags().GetBool("header-only")
			force, _ := cmd.Flags().GetBool("force")
			delimiter, _ := cmd.Flags().GetString("delimiter")

			var comma rune = ','
			if delimiter != "" {
				runes := []rune(delimiter)
				if len(runes) != 1 {
					return fmt.Errorf("--delimiter must be a single character")
				}
				comma = runes[0]
			}

			rows, err := writeUserImportTemplate(filePath, comma, preset, headerOnly, force)
			if err != nil {
				return err
			}

			output, _ := json.Marshal(map[string]any{
				"data": map[string]any{
					"file":        filePath,
					"preset":      preset,
					"header_only": headerOnly,
					"rows":        rows,
				},
			})
			printOutput(opts.Output, output)
			return nil
		},
	}
	importTemplate.Flags().String("file", "", "Output CSV file path")
	importTemplate.Flags().String("preset", "mixed", "Template preset: mixed|local|ldap")
	importTemplate.Flags().String("delimiter", ",", "CSV delimiter, for example , or ;")
	importTemplate.Flags().Bool("header-only", false, "Generate only the CSV header row")
	importTemplate.Flags().Bool("force", false, "Overwrite output file if it already exists")
	_ = importTemplate.MarkFlagRequired("file")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get user by ID, username or email",
		Long:  "Fetch a single user. You can identify the target by exact --id, --username or --email.",
		Example: `sysctl keycloak user get --username test.hieuny
sysctl keycloak user get --email test.hieuny@mbfs.vn
sysctl keycloak user get --id 4e7f9d9a-8e2b-4d5a-9f71-8a6c592a9b11`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			username, _ := cmd.Flags().GetString("username")
			email, _ := cmd.Flags().GetString("email")
			resolvedID, err := resolveUserID(opts, id, username, email)
			if err != nil {
				return err
			}
			b, _, err := api(opts).Do(context.Background(), "GET", "/api/v1/keycloak/users/"+resolvedID, nil)
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	get.Flags().String("id", "", "User ID")
	get.Flags().String("username", "", "Username")
	get.Flags().String("email", "", "Email")

	list := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Example: `sysctl keycloak user list
sysctl keycloak user list --q alice --limit 20 --offset 0`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			q, _ := cmd.Flags().GetString("q")
			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")
			all, _ := cmd.Flags().GetBool("all")
			if !all {
				b, _, err := fetchUsers(opts, q, limit, offset)
				if err != nil {
					return err
				}
				printOutput(opts.Output, b)
				return nil
			}

			if limit <= 0 {
				limit = 200
			}
			client := api(opts)
			type pageResp struct {
				Data []map[string]any `json:"data"`
			}
			allRows := make([]map[string]any, 0)
			var warnings []string
			seen := map[string]struct{}{}
			currentOffset := 0
			for {
				path := buildUserListPath(q, limit, currentOffset)
				b, _, err := client.Do(context.Background(), "GET", path, nil)
				if err != nil {
					return err
				}
				warnings = dedupeWarnings(warnings, extractWarnings(b)...)
				var page pageResp
				if err := json.Unmarshal(b, &page); err != nil {
					return err
				}
				if len(page.Data) == 0 {
					break
				}
				for _, row := range page.Data {
					key := formatCell(row["id"])
					if key == "" {
						key = formatCell(row["username"])
					}
					if key != "" {
						if _, ok := seen[key]; ok {
							continue
						}
						seen[key] = struct{}{}
					}
					allRows = append(allRows, row)
				}
				if len(page.Data) < limit {
					break
				}
				currentOffset += limit
			}

			finalBody, _ := json.Marshal(map[string]any{
				"data":     allRows,
				"warnings": warnings,
			})
			printOutput(opts.Output, finalBody)
			return nil
		},
	}
	list.Flags().String("q", "", "Search query")
	list.Flags().Int("limit", 50, "Limit")
	list.Flags().Int("offset", 0, "Offset")
	list.Flags().Bool("all", false, "Fetch all users by auto-pagination (ignore offset)")

	search := &cobra.Command{
		Use:   "search",
		Short: "Search users",
		Long:  "Search users by substring against Keycloak. Use this to discover the exact username, email, groups and ID before calling get/update/delete.",
		Example: `sysctl keycloak user search --q hieuny
sysctl keycloak user search --q test.hieuny --output table`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			q, _ := cmd.Flags().GetString("q")
			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")
			if strings.TrimSpace(q) == "" {
				return fmt.Errorf("--q is required")
			}
			b, _, err := fetchUsers(opts, q, limit, offset)
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	search.Flags().String("q", "", "Search query")
	search.Flags().Int("limit", 50, "Limit")
	search.Flags().Int("offset", 0, "Offset")

	update := &cobra.Command{
		Use:   "update",
		Short: "Update user",
		Long:  "Update a user identified by exact --id, --username or --email. Only provided fields are changed.",
		Example: `sysctl keycloak user update --username test.hieuny --email alice.new@example.com
sysctl keycloak user update --email test.hieuny@mbfs.vn --enabled=false
sysctl keycloak user update --username test.hieuny --display-name "Alice N" --enabled=false`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			username, _ := cmd.Flags().GetString("username")
			refEmail, _ := cmd.Flags().GetString("ref-email")
			email, _ := cmd.Flags().GetString("email")
			dn, _ := cmd.Flags().GetString("display-name")
			enabled, _ := cmd.Flags().GetString("enabled")
			resolvedID, err := resolveUserID(opts, id, username, refEmail)
			if err != nil {
				return err
			}
			payload := map[string]any{}
			if email != "" {
				payload["email"] = email
			}
			if dn != "" {
				payload["display_name"] = dn
			}
			if enabled != "" {
				enabledValue, err := parseStrictBoolFlag(enabled, "enabled")
				if err != nil {
					return err
				}
				payload["enabled"] = enabledValue
			}
			b, _, err := api(opts).Do(context.Background(), "PUT", "/api/v1/keycloak/users/"+resolvedID, payload)
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	update.Flags().String("id", "", "User ID")
	update.Flags().String("username", "", "Username")
	update.Flags().String("ref-email", "", "Resolve target user by current email")
	update.Flags().String("email", "", "Email")
	update.Flags().String("display-name", "", "Display name")
	update.Flags().String("enabled", "", "true|false")

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable user",
		Long:  "Enable a Keycloak user identified by exact --id, --username or --email.",
		Example: `sysctl keycloak user enable --username test.hieuny
sysctl keycloak user enable --email test.hieuny@mbfs.vn
sysctl keycloak user enable --id <id>`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			username, _ := cmd.Flags().GetString("username")
			email, _ := cmd.Flags().GetString("email")
			resolvedID, err := resolveUserID(opts, id, username, email)
			if err != nil {
				return err
			}
			b, _, err := api(opts).Do(context.Background(), "PUT", "/api/v1/keycloak/users/"+resolvedID, map[string]any{
				"enabled": true,
			})
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	enableCmd.Flags().String("id", "", "User ID")
	enableCmd.Flags().String("username", "", "Username")
	enableCmd.Flags().String("email", "", "Email")

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable user",
		Long:  "Disable a Keycloak user identified by exact --id, --username or --email.",
		Example: `sysctl keycloak user disable --username test.hieuny
sysctl keycloak user disable --email test.hieuny@mbfs.vn
sysctl keycloak user disable --id <id>`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			username, _ := cmd.Flags().GetString("username")
			email, _ := cmd.Flags().GetString("email")
			resolvedID, err := resolveUserID(opts, id, username, email)
			if err != nil {
				return err
			}
			b, _, err := api(opts).Do(context.Background(), "PUT", "/api/v1/keycloak/users/"+resolvedID, map[string]any{
				"enabled": false,
			})
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	disableCmd.Flags().String("id", "", "User ID")
	disableCmd.Flags().String("username", "", "Username")
	disableCmd.Flags().String("email", "", "Email")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete user",
		Long:  "Delete a user identified by exact --id, --username or --email.",
		Example: `sysctl keycloak user delete --username test.hieuny
sysctl keycloak user delete --email test.hieuny@mbfs.vn
sysctl keycloak user delete --id <id>`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			username, _ := cmd.Flags().GetString("username")
			email, _ := cmd.Flags().GetString("email")
			resolvedID, err := resolveUserID(opts, id, username, email)
			if err != nil {
				return err
			}
			_, _, err = api(opts).Do(context.Background(), "DELETE", "/api/v1/keycloak/users/"+resolvedID, nil)
			if err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	deleteCmd.Flags().String("id", "", "User ID")
	deleteCmd.Flags().String("username", "", "Username")
	deleteCmd.Flags().String("email", "", "Email")

	cmd.AddCommand(create, createLocal, createLDAP, importCSV, importTemplate, get, list, search, update, enableCmd, disableCmd, deleteCmd)
	return cmd
}

func keycloakCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keycloak",
		Aliases: []string{"sso", "keycloal"},
		Short:   "Keycloak/SSO operations",
	}
	cmd.AddCommand(userCmd(opts))
	cmd.AddCommand(groupCmd(opts))
	return cmd
}

func groupCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "group", Short: "Manage groups"}

	create := &cobra.Command{
		Use:     "create",
		Short:   "Create group",
		Example: `sysctl keycloak group create --name engineering-vpn`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name, _ := cmd.Flags().GetString("name")
			b, _, err := api(opts).Do(context.Background(), "POST", "/api/v1/keycloak/groups", map[string]any{"name": name})
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	create.Flags().String("name", "", "Group name")
	_ = create.MarkFlagRequired("name")

	get := &cobra.Command{Use: "get", Short: "Get group by ID or name", Long: "Fetch a single group. You can identify the target by exact --id or --name. --name also accepts the full Keycloak group path.", Example: `sysctl keycloak group get --name vanhanh
sysctl keycloak group get --name /vpn/vanhanh
sysctl keycloak group get --id <group-id>`, RunE: func(cmd *cobra.Command, _ []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		resolvedID, err := resolveGroupID(opts, id, name)
		if err != nil {
			return err
		}
		b, _, err := api(opts).Do(context.Background(), "GET", "/api/v1/keycloak/groups/"+resolvedID, nil)
		if err != nil {
			return err
		}
		printOutput(opts.Output, b)
		return nil
	}}
	get.Flags().String("id", "", "Group ID")
	get.Flags().String("name", "", "Group name or full path")

	list := &cobra.Command{Use: "list", Short: "List groups", Example: `sysctl keycloak group list --q eng`, RunE: func(cmd *cobra.Command, _ []string) error {
		q, _ := cmd.Flags().GetString("q")
		b, _, err := fetchGroups(opts, q)
		if err != nil {
			return err
		}
		printOutput(opts.Output, b)
		return nil
	}}
	list.Flags().String("q", "", "Search query")

	update := &cobra.Command{Use: "update", Short: "Update group", Long: "Update a group identified by exact --id or --name. Use --new-name for the new group name.", Example: `sysctl keycloak group update --name vanhanh --new-name van-hanh
sysctl keycloak group update --id <id> --new-name platform-vpn`, RunE: func(cmd *cobra.Command, _ []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		newName, _ := cmd.Flags().GetString("new-name")
		resolvedID, err := resolveGroupID(opts, id, name)
		if err != nil {
			return err
		}
		b, _, err := api(opts).Do(context.Background(), "PUT", "/api/v1/keycloak/groups/"+resolvedID, map[string]any{"name": newName})
		if err != nil {
			return err
		}
		printOutput(opts.Output, b)
		return nil
	}}
	update.Flags().String("id", "", "Group ID")
	update.Flags().String("name", "", "Current group name or full path")
	update.Flags().String("new-name", "", "New group name")
	_ = update.MarkFlagRequired("new-name")

	deleteCmd := &cobra.Command{Use: "delete", Short: "Delete group", Long: "Delete a group identified by exact --id or --name.", Example: `sysctl keycloak group delete --name vanhanh
sysctl keycloak group delete --id <id>`, RunE: func(cmd *cobra.Command, _ []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		resolvedID, err := resolveGroupID(opts, id, name)
		if err != nil {
			return err
		}
		_, _, err = api(opts).Do(context.Background(), "DELETE", "/api/v1/keycloak/groups/"+resolvedID, nil)
		if err != nil {
			return err
		}
		fmt.Println("deleted")
		return nil
	}}
	deleteCmd.Flags().String("id", "", "Group ID")
	deleteCmd.Flags().String("name", "", "Group name or full path")

	addMember := &cobra.Command{Use: "add-member", Short: "Add member", Long: "Add a user to a group identified by exact --id or --name.", Example: `sysctl keycloak group add-member --name vanhanh --user-id <user-id>
sysctl keycloak group add-member --name /vpn/vanhanh --user-id <user-id>
sysctl keycloak group add-member --id <group-id> --user-id <user-id>`, RunE: func(cmd *cobra.Command, _ []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		uid, _ := cmd.Flags().GetString("user-id")
		gid, err := resolveGroupID(opts, id, name)
		if err != nil {
			return err
		}
		_, _, err = api(opts).Do(context.Background(), "POST", "/api/v1/keycloak/groups/"+gid+"/members", map[string]any{"user_id": uid})
		if err != nil {
			return err
		}
		fmt.Println("ok")
		return nil
	}}
	addMember.Flags().String("id", "", "Group ID")
	addMember.Flags().String("name", "", "Group name or full path")
	addMember.Flags().String("user-id", "", "User ID")
	_ = addMember.MarkFlagRequired("user-id")

	removeMember := &cobra.Command{Use: "remove-member", Short: "Remove member", Long: "Remove a user from a group identified by exact --id or --name.", Example: `sysctl keycloak group remove-member --name vanhanh --user-id <user-id>
sysctl keycloak group remove-member --name /vpn/vanhanh --user-id <user-id>
sysctl keycloak group remove-member --id <group-id> --user-id <user-id>`, RunE: func(cmd *cobra.Command, _ []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		uid, _ := cmd.Flags().GetString("user-id")
		gid, err := resolveGroupID(opts, id, name)
		if err != nil {
			return err
		}
		_, _, err = api(opts).Do(context.Background(), "DELETE", "/api/v1/keycloak/groups/"+gid+"/members/"+uid, nil)
		if err != nil {
			return err
		}
		fmt.Println("ok")
		return nil
	}}
	removeMember.Flags().String("id", "", "Group ID")
	removeMember.Flags().String("name", "", "Group name or full path")
	removeMember.Flags().String("user-id", "", "User ID")
	_ = removeMember.MarkFlagRequired("user-id")

	cmd.AddCommand(create, get, list, update, deleteCmd, addMember, removeMember)
	return cmd
}

func buildOpenVPNAccessListCommand(opts *rootOptions, subjectType string) *cobra.Command {
	subjectFlag := "username"
	subjectLabel := "user"
	subjectExample := "test.hieuny"
	basePath := "/api/v1/openvpn/users"
	switch subjectType {
	case "user":
	case "group":
		subjectFlag = "groupname"
		subjectLabel = "group"
		subjectExample = "vanhanh"
		basePath = "/api/v1/openvpn/groups"
	default:
		panic("unsupported openvpn access-list subject type")
	}

	accessList := &cobra.Command{
		Use:   "access-list",
		Short: fmt.Sprintf("Access-list lifecycle for %s", subjectLabel),
	}

	listAccess := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List unified access entries for one %s", subjectLabel),
		Long:  fmt.Sprintf("List the effective OpenVPN entries for exactly one %s. The table merges IP/CIDR access-list rows with domain-routing rules so you can inspect one owner in a single place.", subjectLabel),
		Example: fmt.Sprintf(
			"sysctl --output table openvpn %s access-list list --%s %s",
			subjectType, subjectFlag, subjectExample,
		),
		RunE: func(cmd *cobra.Command, _ []string) error {
			subject, err := resolveAccessListSubject(cmd, subjectType)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("%s/%s/access-lists", basePath, url.PathEscape(subject))
			b, _, err := api(opts).Do(context.Background(), "GET", path, nil)
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	listAccess.Flags().String(subjectFlag, "", fmt.Sprintf("Filter by %s", subjectFlag))
	_ = listAccess.MarkFlagRequired(subjectFlag)
	accessList.AddCommand(listAccess)

	for _, mode := range []string{"append", "remove"} {
		mode := mode
		modeTitle := "Append"
		longText := "Use one payload for both IP/CIDR and domain entries. `target` auto-detects IP/CIDR -> access list, otherwise domain -> rules. Omit `--port` or set `--port all` to apply to all ports."
		if mode == "append" {
			longText += " `append` means add or update only the submitted entry without removing other existing entries. Do not mix IP/CIDR and domain targets in the same request. For user domain targets, the backend can bootstrap a ruleset if missing. For group domain targets, the group must already have an assigned ruleset in OpenVPN AS."
		} else {
			modeTitle = "Remove"
			longText += " `remove` means delete only the matching submitted entry. Do not mix IP/CIDR and domain targets in the same request."
		}

		c := &cobra.Command{
			Use:   mode,
			Short: modeTitle + " unified access entries",
			Example: fmt.Sprintf(`sysctl openvpn %[1]s access-list %[2]s --%[3]s %[4]s --target 10.0.0.0/8
sysctl openvpn %[1]s access-list %[2]s --%[3]s %[4]s --target ndc.dichvucong.gov.vn
sysctl openvpn %[1]s access-list %[2]s --%[3]s %[4]s --file access-list.json`, subjectType, mode, subjectFlag, subjectExample),
			Long: longText,
			RunE: func(cmd *cobra.Command, _ []string) error {
				subject, err := resolveAccessListSubject(cmd, subjectType)
				if err != nil {
					return err
				}
				body, err := buildAccessListBodyFromFlags(cmd, subjectType, subject)
				if err != nil {
					return err
				}
				path := fmt.Sprintf("%s/%s/access-lists:%s", basePath, url.PathEscape(subject), mode)
				_, _, err = api(opts).Do(context.Background(), "POST", path, body)
				return err
			},
		}
		c.Flags().String("file", "", "JSON payload file")
		c.Flags().String(subjectFlag, "", fmt.Sprintf("Apply to %s", subjectFlag))
		c.Flags().String("target", "", "Target domain or IP/CIDR")
		c.Flags().String("port", "", "Port filter for IP/CIDR entries, for example tcp:443 or udp:53; omit or set all for all ports")
		_ = c.MarkFlagRequired(subjectFlag)
		accessList.AddCommand(c)
	}

	return accessList
}

func openvpnCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "openvpn", Short: "OpenVPN operations"}

	ovpnUsers := &cobra.Command{Use: "user", Short: "OpenVPN users"}
	ovpnUserList := &cobra.Command{Use: "list", Short: "List OpenVPN users", Example: `sysctl openvpn user list
sysctl openvpn user list --q test --limit 100 --offset 0
sysctl openvpn user list --all --output table`, RunE: func(cmd *cobra.Command, _ []string) error {
		q, _ := cmd.Flags().GetString("q")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		all, _ := cmd.Flags().GetBool("all")
		if all {
			offset = 0
		}
		params := url.Values{}
		params.Set("q", q)
		params.Set("limit", strconv.Itoa(limit))
		params.Set("offset", strconv.Itoa(offset))
		params.Set("all", strconv.FormatBool(all))
		path := "/api/v1/openvpn/users?" + params.Encode()
		b, _, err := api(opts).Do(context.Background(), "GET", path, nil)
		if err != nil {
			return err
		}
		printOutput(opts.Output, b)
		return nil
	}}
	ovpnUserList.Flags().String("q", "", "Search by username substring")
	ovpnUserList.Flags().Int("limit", 50, "Limit")
	ovpnUserList.Flags().Int("offset", 0, "Offset")
	ovpnUserList.Flags().Bool("all", false, "Fetch all users by auto-pagination (ignore offset)")
	ovpnUsers.AddCommand(ovpnUserList)

	ovpnUserCreateFromKeycloak := &cobra.Command{
		Use:   "create-from-keycloak",
		Short: "Create OpenVPN user from Keycloak",
		Long:  "Provision an OpenVPN user by first loading identity data from Keycloak, then calling OpenVPN APIs. Use this explicit bridge command when you want to combine SSO and OpenVPN.",
		Example: `sysctl openvpn user create-from-keycloak --username test.local
sysctl openvpn user create-from-keycloak --user-id <keycloak-user-id> --vpn-group partners-vpn`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			userID, _ := cmd.Flags().GetString("user-id")
			username, _ := cmd.Flags().GetString("username")
			vpnGroup, _ := cmd.Flags().GetString("vpn-group")
			if strings.TrimSpace(userID) == "" && strings.TrimSpace(username) == "" {
				return fmt.Errorf("set either --user-id or --username")
			}
			payload := map[string]any{}
			if strings.TrimSpace(userID) != "" {
				payload["user_id"] = strings.TrimSpace(userID)
			}
			if strings.TrimSpace(username) != "" {
				payload["username"] = strings.TrimSpace(username)
			}
			if strings.TrimSpace(vpnGroup) != "" {
				payload["vpn_group"] = strings.TrimSpace(vpnGroup)
			}
			b, _, err := api(opts).Do(context.Background(), "POST", "/api/v1/openvpn/users:create-from-keycloak", payload)
			if err != nil {
				return err
			}
			printOutput(opts.Output, b)
			return nil
		},
	}
	ovpnUserCreateFromKeycloak.Flags().String("user-id", "", "Keycloak user ID")
	ovpnUserCreateFromKeycloak.Flags().String("username", "", "Keycloak username")
	ovpnUserCreateFromKeycloak.Flags().String("vpn-group", "", "OpenVPN group to assign")
	ovpnUsers.AddCommand(ovpnUserCreateFromKeycloak)

	ovpnUserExport := &cobra.Command{Use: "export", Short: "Export OpenVPN users", Long: "Export OpenVPN users from OpenVPN API. Without --file, the command prints JSON/table to stdout. With --file, it writes a CSV file.", Example: `sysctl openvpn user export --file openvpn-users.csv
sysctl openvpn user export --q test --output table`, RunE: func(cmd *cobra.Command, _ []string) error {
		q, _ := cmd.Flags().GetString("q")
		filePath, _ := cmd.Flags().GetString("file")
		params := url.Values{}
		params.Set("q", q)
		path := "/api/v1/openvpn/users:export?" + params.Encode()
		b, _, err := api(opts).Do(context.Background(), "GET", path, nil)
		if err != nil {
			return err
		}

		if strings.TrimSpace(filePath) == "" {
			printOutput(opts.Output, b)
			return nil
		}
		printWarnings(extractWarnings(b))

		var resp struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			return err
		}
		if err := writeOpenVPNUserExportCSV(filePath, resp.Data); err != nil {
			return err
		}
		fmt.Printf("exported %d users to %s\n", len(resp.Data), filePath)
		return nil
	}}
	ovpnUserExport.Flags().String("q", "", "Search by username substring before export")
	ovpnUserExport.Flags().String("file", "", "CSV output file path")
	ovpnUsers.AddCommand(ovpnUserExport)
	ovpnUsers.AddCommand(buildOpenVPNAccessListCommand(opts, "user"))

	ovpnGroups := &cobra.Command{Use: "group", Short: "OpenVPN groups"}
	ovpnGroupList := &cobra.Command{Use: "list", Short: "List OpenVPN groups", Example: `sysctl openvpn group list
sysctl openvpn group list --q vpn --enumerate-members
sysctl openvpn group list --all --output table`, RunE: func(cmd *cobra.Command, _ []string) error {
		q, _ := cmd.Flags().GetString("q")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		enumerateMembers, _ := cmd.Flags().GetBool("enumerate-members")
		all, _ := cmd.Flags().GetBool("all")
		if all {
			offset = 0
		}
		params := url.Values{}
		params.Set("q", q)
		params.Set("limit", strconv.Itoa(limit))
		params.Set("offset", strconv.Itoa(offset))
		params.Set("enumerate_members", strconv.FormatBool(enumerateMembers))
		params.Set("all", strconv.FormatBool(all))
		path := "/api/v1/openvpn/groups?" + params.Encode()
		b, _, err := api(opts).Do(context.Background(), "GET", path, nil)
		if err != nil {
			return err
		}
		printOutput(opts.Output, b)
		return nil
	}}
	ovpnGroupList.Flags().String("q", "", "Search by groupname substring")
	ovpnGroupList.Flags().Int("limit", 50, "Limit")
	ovpnGroupList.Flags().Int("offset", 0, "Offset")
	ovpnGroupList.Flags().Bool("all", false, "Fetch all groups by auto-pagination (ignore offset)")
	ovpnGroupList.Flags().Bool("enumerate-members", false, "Include group members list")
	ovpnGroups.AddCommand(ovpnGroupList)
	ovpnGroups.AddCommand(buildOpenVPNAccessListCommand(opts, "group"))

	cmd.AddCommand(ovpnUsers, ovpnGroups)
	return cmd
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
