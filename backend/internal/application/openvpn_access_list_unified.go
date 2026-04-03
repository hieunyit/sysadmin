package application

import (
	"context"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	domainerr "backend/internal/domain/errors"
	"backend/internal/domain/services"
)

type accessSubjectKey struct {
	SubjectType string
	Subject     string
}

type normalizedAccessEntry struct {
	SubjectType string
	Subject     string
	Kind        string
	Direction   string
	RouteType   string
	Accept      bool
	CIDR        string
	ServiceSpec string
	Domain      string
	MatchType   string
	Action      string
	Position    int
	HasPosition bool
	Comment     string
}

type existingOpenVPNRule struct {
	ID        int64
	RulesetID int64
	Type      string
	MatchType string
	MatchData string
	Action    string
	Position  int
	Comment   string
}

type domainMutationPlan struct {
	Owner     accessSubjectKey
	Ruleset   services.OpenVPNRuleset
	Add       []services.OpenVPNRule
	DeleteIDs []int64
}

func (s *OpenVPNAdminService) ListAccessEntries(ctx context.Context, q services.OpenVPNAccessListQuery) (map[string]any, error) {
	accessOut, err := s.ovpn.ListAccessLists(ctx, q)
	if err != nil {
		return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn access-list list failed", err)
	}

	items := flattenAccessProfiles(accessOut)
	rawRulesets := make([]any, 0)
	rawRules := make([]any, 0)

	for _, owner := range accessQueryOwners(q) {
		rulesets, err := s.ovpn.ListRulesets(ctx, owner.Subject, "")
		if err != nil {
			return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn rulesets list by owner failed", err)
		}
		if len(rulesets) == 0 {
			continue
		}
		ids := make([]int64, 0, len(rulesets))
		rulesetIndex := make(map[int64]services.OpenVPNRuleset, len(rulesets))
		for _, rs := range rulesets {
			ids = append(ids, rs.ID)
			rulesetIndex[rs.ID] = rs
			rawRulesets = append(rawRulesets, map[string]any{
				"id":         rs.ID,
				"name":       rs.Name,
				"comment":    rs.Comment,
				"owner":      rs.Owner,
				"owner_type": rs.OwnerType,
				"position":   rs.Position,
			})
		}

		rulesOut, err := s.ovpn.ListRules(ctx, services.OpenVPNRuleListQuery{
			RulesetIDs: ids,
			Type:       "domain_routing",
		})
		if err != nil {
			return nil, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn rules list failed", err)
		}
		domainItems, domainRules := flattenDomainRules(rulesOut, rulesetIndex, owner)
		items = append(items, domainItems...)
		rawRules = append(rawRules, domainRules...)
	}

	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if asString(left["subject_type"]) != asString(right["subject_type"]) {
			return asString(left["subject_type"]) < asString(right["subject_type"])
		}
		if asString(left["subject"]) != asString(right["subject"]) {
			return asString(left["subject"]) < asString(right["subject"])
		}
		if asString(left["kind"]) != asString(right["kind"]) {
			return asString(left["kind"]) < asString(right["kind"])
		}
		if asString(left["target"]) != asString(right["target"]) {
			return asString(left["target"]) < asString(right["target"])
		}
		return asString(left["entry_type"]) < asString(right["entry_type"])
	})

	return map[string]any{
		"items":    items,
		"profiles": accessOut["profiles"],
		"rulesets": rawRulesets,
		"rules":    rawRules,
		"total":    len(items),
	}, nil
}

func (s *OpenVPNAdminService) ApplyAccessEntries(ctx context.Context, mode string, items []services.OpenVPNAccessEntryInput, actor string) error {
	if len(items) == 0 {
		return domainerr.New(domainerr.CodeInvalidArgument, "items are required")
	}

	normalized, err := normalizeAccessEntries(items)
	if err != nil {
		return domainerr.Wrap(domainerr.CodeInvalidArgument, "invalid access-list payload", err)
	}

	accessItems := make([]services.AccessRouteItem, 0, len(normalized))
	domainByOwner := make(map[accessSubjectKey][]normalizedAccessEntry)
	for _, item := range normalized {
		switch item.Kind {
		case "access":
			route := services.AccessRouteItem{
				Type:      item.Direction,
				RouteType: item.RouteType,
				Accept:    item.Accept,
			}
			if item.SubjectType == "user" {
				name := item.Subject
				route.Username = &name
			} else {
				name := item.Subject
				route.Groupname = &name
			}
			if item.CIDR != "" {
				cidr := item.CIDR
				route.CIDR = &cidr
			}
			if item.ServiceSpec != "" {
				spec := item.ServiceSpec
				route.ServiceSpec = &spec
			}
			accessItems = append(accessItems, route)
		case "domain":
			key := accessSubjectKey{SubjectType: item.SubjectType, Subject: item.Subject}
			domainByOwner[key] = append(domainByOwner[key], item)
		default:
			return domainerr.New(domainerr.CodeInvalidArgument, "unsupported access-list item kind")
		}
	}

	if len(accessItems) > 0 && len(domainByOwner) > 0 {
		return domainerr.New(domainerr.CodeInvalidArgument, "mixed IP/CIDR and domain changes are not supported in one request; run separate access-list commands")
	}
	if len(domainByOwner) > 1 {
		return domainerr.New(domainerr.CodeInvalidArgument, "domain changes must target exactly one user or group per request")
	}

	plans := make([]domainMutationPlan, 0, len(domainByOwner))
	for owner, domainItems := range domainByOwner {
		plan, err := s.prepareDomainMutation(ctx, mode, owner, domainItems)
		if err != nil {
			return err
		}
		if len(plan.Add) == 0 && len(plan.DeleteIDs) == 0 {
			continue
		}
		plans = append(plans, plan)
	}

	for _, plan := range plans {
		if _, err := s.ovpn.ModifyRules(ctx, plan.Add, plan.DeleteIDs); err != nil {
			return domainerr.Wrap(domainerr.CodeExternalFailure, fmt.Sprintf("openvpn domain rule apply failed for %s %s", plan.Owner.SubjectType, plan.Owner.Subject), err)
		}
	}

	if len(accessItems) > 0 {
		switch mode {
		case "set":
			err = s.ovpn.SetAccessList(ctx, accessItems)
		case "append":
			err = s.ovpn.AppendAccessList(ctx, accessItems)
		case "remove":
			err = s.ovpn.RemoveAccessList(ctx, accessItems)
		default:
			err = domainerr.New(domainerr.CodeInvalidArgument, "unsupported access-list mode")
		}
		if err != nil {
			return domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn access-list apply failed", err)
		}
	}

	if s.notifications != nil {
		s.notifyAccessChange(ctx, mode, normalized, actor)
	}

	return nil
}

func (s *OpenVPNAdminService) notifyAccessChange(ctx context.Context, mode string, items []normalizedAccessEntry, actor string) {
	if s.notifications == nil || len(items) == 0 {
		return
	}

	byOwner := make(map[accessSubjectKey][]normalizedAccessEntry)
	for _, item := range items {
		key := accessSubjectKey{
			SubjectType: strings.TrimSpace(item.SubjectType),
			Subject:     strings.TrimSpace(item.Subject),
		}
		if key.SubjectType == "" || key.Subject == "" {
			continue
		}
		byOwner[key] = append(byOwner[key], item)
	}

	for owner, ownerItems := range byOwner {
		recipients, err := s.resolveNotificationRecipients(ctx, owner)
		if err != nil || len(recipients) == 0 {
			continue
		}
		s.notifications.TrySendVPNAccessChanged(ctx, recipients, owner.SubjectType, owner.Subject, mode, actor, buildAccessNotificationEntries(ownerItems))
	}
}

func (s *OpenVPNAdminService) resolveNotificationRecipients(ctx context.Context, owner accessSubjectKey) ([]services.KeycloakUser, error) {
	_ = ctx
	_ = owner
	return nil, nil
}

func buildAccessNotificationEntries(items []normalizedAccessEntry) []accessNotificationEntry {
	if len(items) == 0 {
		return nil
	}
	out := make([]accessNotificationEntry, 0, len(items))
	for _, item := range items {
		switch item.Kind {
		case "domain":
			out = append(out, accessNotificationEntry{
				Summary: fmt.Sprintf("Domain %s | Hành động: %s", strings.TrimSpace(item.Domain), strings.TrimSpace(item.Action)),
			})
		case "access":
			port := strings.TrimSpace(item.ServiceSpec)
			if port == "" {
				port = "tất cả"
			}
			out = append(out, accessNotificationEntry{
				Summary: fmt.Sprintf("IP %s | Cổng: %s | Kiểu route: %s", strings.TrimSpace(item.CIDR), port, strings.TrimSpace(item.RouteType)),
			})
		}
	}
	return out
}

func (s *OpenVPNAdminService) prepareDomainMutation(ctx context.Context, mode string, owner accessSubjectKey, items []normalizedAccessEntry) (domainMutationPlan, error) {
	plan := domainMutationPlan{Owner: owner}

	rulesets, err := s.ovpn.ListRulesets(ctx, owner.Subject, "")
	if err != nil {
		return plan, domainerr.Wrap(domainerr.CodeExternalFailure, fmt.Sprintf("openvpn rulesets list failed for %s", owner.Subject), err)
	}
	if len(rulesets) == 0 {
		if owner.SubjectType == "group" {
			return plan, domainerr.New(
				domainerr.CodePreconditionFail,
				fmt.Sprintf("group %q has no OpenVPN ruleset for domain routing", owner.Subject),
			)
		}
		bootstrapped, err := s.ensureRulesetForOwner(ctx, owner)
		if err != nil {
			return plan, err
		}
		rulesets = []services.OpenVPNRuleset{bootstrapped}
	}
	if len(rulesets) > 1 {
		return plan, domainerr.New(domainerr.CodePreconditionFail, fmt.Sprintf("%s %s has %d rulesets; unified access-list domain management requires exactly one ruleset", owner.SubjectType, owner.Subject, len(rulesets)))
	}
	plan.Ruleset = rulesets[0]

	rawRules, err := s.ovpn.ListRules(ctx, services.OpenVPNRuleListQuery{
		RulesetIDs: []int64{plan.Ruleset.ID},
		Type:       "domain_routing",
	})
	if err != nil {
		return plan, domainerr.Wrap(domainerr.CodeExternalFailure, "openvpn rules list failed", err)
	}
	existing := parseExistingRules(rawRules)
	existingByKey := make(map[string]existingOpenVPNRule, len(existing))
	for _, rule := range existing {
		existingByKey[domainRuleKey(rule.MatchType, rule.MatchData)] = rule
	}

	desired := make(map[string]normalizedAccessEntry, len(items))
	for _, item := range items {
		desired[domainRuleKey(item.MatchType, item.Domain)] = item
	}

	switch mode {
	case "append":
		nextPosition := maxExistingRulePosition(existing) + 1
		for _, item := range items {
			key := domainRuleKey(item.MatchType, item.Domain)
			rule := buildOpenVPNRule(plan.Ruleset.ID, item)
			if existingRule, ok := existingByKey[key]; ok {
				rule.ID = &existingRule.ID
				if !item.HasPosition {
					rule.Position = existingRule.Position
				}
			} else if !item.HasPosition {
				rule.Position = nextPosition
				nextPosition++
			}
			plan.Add = append(plan.Add, rule)
		}
	case "remove":
		for _, item := range items {
			key := domainRuleKey(item.MatchType, item.Domain)
			if existingRule, ok := existingByKey[key]; ok {
				plan.DeleteIDs = append(plan.DeleteIDs, existingRule.ID)
			}
		}
	case "set":
		nextPosition := maxExistingRulePosition(existing) + 1
		for _, item := range items {
			key := domainRuleKey(item.MatchType, item.Domain)
			rule := buildOpenVPNRule(plan.Ruleset.ID, item)
			if existingRule, ok := existingByKey[key]; ok {
				rule.ID = &existingRule.ID
				if !item.HasPosition {
					rule.Position = existingRule.Position
				}
			} else if !item.HasPosition {
				rule.Position = nextPosition
				nextPosition++
			}
			plan.Add = append(plan.Add, rule)
		}
		for key, existingRule := range existingByKey {
			if _, ok := desired[key]; ok {
				continue
			}
			plan.DeleteIDs = append(plan.DeleteIDs, existingRule.ID)
		}
	default:
		return plan, domainerr.New(domainerr.CodeInvalidArgument, "unsupported access-list mode")
	}

	return plan, nil
}

func buildOpenVPNRule(rulesetID int64, item normalizedAccessEntry) services.OpenVPNRule {
	return services.OpenVPNRule{
		RulesetID: rulesetID,
		Type:      "domain_routing",
		MatchType: item.MatchType,
		MatchData: item.Domain,
		Action:    item.Action,
		Position:  item.Position,
		Comment:   item.Comment,
	}
}

func normalizeAccessEntries(items []services.OpenVPNAccessEntryInput) ([]normalizedAccessEntry, error) {
	out := make([]normalizedAccessEntry, 0, len(items))
	domainPosition := make(map[accessSubjectKey]int)

	for _, item := range items {
		subject, err := resolveAccessSubject(item.Username, item.Groupname)
		if err != nil {
			return nil, err
		}

		if domain := firstNonEmpty(item.Domain); domain != "" {
			out = append(out, normalizeDomainEntry(item, subject, strings.TrimSpace(domain), domainPosition))
			continue
		}

		if target := firstNonEmpty(item.Target); target != "" {
			if cidr, ok := normalizeAsCIDR(target); ok {
				out = append(out, normalizeAccessEntry(item, subject, cidr))
				continue
			}
			out = append(out, normalizeDomainEntry(item, subject, strings.TrimSpace(target), domainPosition))
			continue
		}

		if cidr := firstNonEmpty(item.CIDR); cidr != "" {
			normalizedCIDR, ok := normalizeAsCIDR(cidr)
			if !ok {
				return nil, fmt.Errorf("invalid cidr/ip: %s", cidr)
			}
			out = append(out, normalizeAccessEntry(item, subject, normalizedCIDR))
			continue
		}

		return nil, fmt.Errorf("each item must include domain, cidr or target")
	}

	return out, nil
}

func normalizeDomainEntry(item services.OpenVPNAccessEntryInput, subject accessSubjectKey, domain string, positions map[accessSubjectKey]int) normalizedAccessEntry {
	position := 0
	hasPosition := false
	if item.Position != nil {
		position = *item.Position
		hasPosition = true
	}
	if position <= 0 && item.Position != nil {
		hasPosition = false
	}
	if !hasPosition {
		positions[subject]++
		position = positions[subject]
	}

	matchType := strings.TrimSpace(item.MatchType)
	if matchType == "" {
		matchType = "domain"
	}
	action := strings.TrimSpace(item.Action)
	if action == "" {
		action = "nat"
	}

	return normalizedAccessEntry{
		SubjectType: subject.SubjectType,
		Subject:     subject.Subject,
		Kind:        "domain",
		Domain:      strings.ToLower(domain),
		MatchType:   matchType,
		Action:      action,
		Position:    position,
		HasPosition: hasPosition,
		Comment:     item.Comment,
	}
}

func normalizeAccessEntry(item services.OpenVPNAccessEntryInput, subject accessSubjectKey, cidr string) normalizedAccessEntry {
	direction := strings.TrimSpace(item.Type)
	if direction == "" {
		if strings.Contains(cidr, ":") {
			direction = "access_to_ipv6"
		} else {
			direction = "access_to_ipv4"
		}
	}

	routeType := strings.TrimSpace(item.RouteType)
	if routeType == "" {
		routeType = "nat"
	}

	accept := true
	if item.Accept != nil {
		accept = *item.Accept
	}

	return normalizedAccessEntry{
		SubjectType: subject.SubjectType,
		Subject:     subject.Subject,
		Kind:        "access",
		Direction:   direction,
		RouteType:   routeType,
		Accept:      accept,
		CIDR:        cidr,
		ServiceSpec: strings.TrimSpace(firstNonEmpty(item.ServiceSpec)),
	}
}

func resolveAccessSubject(username, groupname *string) (accessSubjectKey, error) {
	user := strings.TrimSpace(firstNonEmpty(username))
	group := strings.TrimSpace(firstNonEmpty(groupname))
	switch {
	case user != "" && group != "":
		return accessSubjectKey{}, fmt.Errorf("set either username or groupname for each item")
	case user != "":
		return accessSubjectKey{SubjectType: "user", Subject: user}, nil
	case group != "":
		return accessSubjectKey{SubjectType: "group", Subject: group}, nil
	default:
		return accessSubjectKey{}, fmt.Errorf("each item must include username or groupname")
	}
}

func accessQueryOwners(q services.OpenVPNAccessListQuery) []accessSubjectKey {
	owners := make([]accessSubjectKey, 0, 2)
	if username := strings.TrimSpace(q.Username); username != "" {
		owners = append(owners, accessSubjectKey{SubjectType: "user", Subject: username})
	}
	if groupname := strings.TrimSpace(q.Groupname); groupname != "" {
		owners = append(owners, accessSubjectKey{SubjectType: "group", Subject: groupname})
	}
	return owners
}

func flattenAccessProfiles(out map[string]any) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, raw := range asSlice(out["profiles"]) {
		row := asMap(raw)
		if row == nil {
			continue
		}
		username := strings.TrimSpace(asString(row["username"]))
		groupname := strings.TrimSpace(asString(row["groupname"]))
		subjectType := ""
		subject := ""
		switch {
		case username != "":
			subjectType = "user"
			subject = username
		case groupname != "":
			subjectType = "group"
			subject = groupname
		}

		route := asMap(row["access_route"])
		target := ""
		port := "all"
		action := ""
		accept := ""
		if route != nil {
			action = asString(route["type"])
			accept = asString(route["accept"])
			subnet := asMap(route["subnet"])
			if subnet != nil {
				netip := asString(subnet["netip"])
				prefix := asInt(subnet["prefix_length"])
				if netip != "" && prefix >= 0 {
					target = fmt.Sprintf("%s/%d", netip, prefix)
				}
				port = joinServices(asSlice(subnet["service"]))
				if port == "" {
					port = "all"
				}
			}
			if target == "" {
				if username := asString(route["username"]); username != "" {
					target = username
				}
				if groupname := asString(route["groupname"]); groupname != "" {
					target = groupname
				}
			}
		}

		rows = append(rows, map[string]any{
			"subject_type": subjectType,
			"subject":      subject,
			"kind":         "access",
			"entry_type":   asString(row["type"]),
			"target":       target,
			"action":       action,
			"accept":       accept,
			"port":         port,
			"comment":      "",
		})
	}
	return rows
}

func flattenDomainRules(out map[string]any, rulesets map[int64]services.OpenVPNRuleset, owner accessSubjectKey) ([]map[string]any, []any) {
	items := make([]map[string]any, 0)
	rawRules := make([]any, 0)
	for _, raw := range asSlice(out["rules"]) {
		row := asMap(raw)
		if row == nil {
			continue
		}
		rawRules = append(rawRules, row)
		rulesetID := asInt64(row["ruleset_id"])
		rs, ok := rulesets[rulesetID]
		subject := owner.Subject
		subjectType := owner.SubjectType
		if ok {
			if strings.TrimSpace(rs.Owner) != "" {
				subject = rs.Owner
			}
			if strings.TrimSpace(rs.OwnerType) != "" {
				subjectType = rs.OwnerType
			}
		}
		items = append(items, map[string]any{
			"subject_type": subjectType,
			"subject":      subject,
			"kind":         "domain",
			"entry_type":   asString(row["type"]),
			"target":       asString(row["match_data"]),
			"action":       asString(row["action"]),
			"accept":       "",
			"port":         "",
			"comment":      asString(row["comment"]),
		})
	}
	return items, rawRules
}

func parseExistingRules(out map[string]any) []existingOpenVPNRule {
	rules := make([]existingOpenVPNRule, 0)
	for _, raw := range asSlice(out["rules"]) {
		row := asMap(raw)
		if row == nil {
			continue
		}
		rules = append(rules, existingOpenVPNRule{
			ID:        asInt64(row["id"]),
			RulesetID: asInt64(row["ruleset_id"]),
			Type:      asString(row["type"]),
			MatchType: asString(row["match_type"]),
			MatchData: strings.ToLower(asString(row["match_data"])),
			Action:    asString(row["action"]),
			Position:  asInt(row["position"]),
			Comment:   asString(row["comment"]),
		})
	}
	return rules
}

func domainRuleKey(matchType, domain string) string {
	return strings.TrimSpace(matchType) + "\x00" + strings.ToLower(strings.TrimSpace(domain))
}

func normalizeAsCIDR(input string) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}
	if prefix, err := netip.ParsePrefix(input); err == nil {
		return prefix.String(), true
	}
	if addr, err := netip.ParseAddr(input); err == nil {
		if addr.Is6() {
			return netip.PrefixFrom(addr, 128).String(), true
		}
		return netip.PrefixFrom(addr, 32).String(), true
	}
	return "", false
}

func firstNonEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func asSlice(v any) []any {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	return items
}

func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

func asInt64(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	default:
		return 0
	}
}

func joinServices(items []any) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, raw := range items {
		service := asMap(raw)
		if service == nil {
			continue
		}
		protocol := asString(service["protocol"])
		startPort := asString(service["start_port"])
		endPort := asString(service["end_port"])
		switch {
		case protocol != "" && startPort != "" && endPort != "":
			parts = append(parts, fmt.Sprintf("%s:%s-%s", protocol, startPort, endPort))
		case protocol != "" && startPort != "":
			parts = append(parts, fmt.Sprintf("%s:%s", protocol, startPort))
		case protocol != "":
			parts = append(parts, protocol)
		}
	}
	return strings.Join(parts, ",")
}

func maxExistingRulePosition(items []existingOpenVPNRule) int {
	maxPos := 0
	for _, item := range items {
		if item.Position > maxPos {
			maxPos = item.Position
		}
	}
	return maxPos
}

func (s *OpenVPNAdminService) ensureRulesetForOwner(ctx context.Context, owner accessSubjectKey) (services.OpenVPNRuleset, error) {
	switch owner.SubjectType {
	case "user":
		if err := s.ovpn.EnsureUser(ctx, owner.Subject); err != nil {
			return services.OpenVPNRuleset{}, domainerr.Wrap(domainerr.CodeExternalFailure, fmt.Sprintf("ensure openvpn user %s failed", owner.Subject), err)
		}
	case "group":
		return services.OpenVPNRuleset{}, domainerr.New(
			domainerr.CodePreconditionFail,
			fmt.Sprintf("group %q has no OpenVPN ruleset for domain routing", owner.Subject),
		)
	default:
		return services.OpenVPNRuleset{}, domainerr.New(domainerr.CodeInvalidArgument, "unsupported owner type")
	}

	name := fmt.Sprintf("Ruleset for %s", owner.Subject)
	comment := fmt.Sprintf("Auto-created ruleset for %s %s", owner.SubjectType, owner.Subject)
	rulesetID, err := s.ovpn.AddRuleset(ctx, name, comment)
	if err != nil {
		return services.OpenVPNRuleset{}, domainerr.Wrap(domainerr.CodeExternalFailure, fmt.Sprintf("create openvpn ruleset for %s failed", owner.Subject), err)
	}

	rollback := true
	defer func() {
		if rollback {
			_ = s.ovpn.DeleteRulesets(ctx, []int64{rulesetID})
		}
	}()

	if err := s.ovpn.ModifyUserRulesetMapping(ctx, map[string][]services.SubjectRulesetRef{
		owner.Subject: {
			{
				RulesetID: rulesetID,
				Position:  1,
			},
		},
	}, nil); err != nil {
		return services.OpenVPNRuleset{}, domainerr.Wrap(domainerr.CodeExternalFailure, fmt.Sprintf("attach openvpn ruleset to %s %s failed", owner.SubjectType, owner.Subject), err)
	}

	rulesets, err := s.ovpn.ListRulesets(ctx, owner.Subject, "")
	if err != nil {
		return services.OpenVPNRuleset{}, domainerr.Wrap(domainerr.CodeExternalFailure, fmt.Sprintf("verify openvpn ruleset for %s failed", owner.Subject), err)
	}
	for _, ruleset := range rulesets {
		if ruleset.ID == rulesetID {
			rollback = false
			return ruleset, nil
		}
	}

	return services.OpenVPNRuleset{}, domainerr.New(domainerr.CodePreconditionFail, fmt.Sprintf("no OpenVPN ruleset found for %s %s; auto-bootstrap could not verify owner attachment", owner.SubjectType, owner.Subject))
}
