package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultOpenCodeGoBaseURL           = "https://opencode.ai/zen/go/v1"
	DefaultOpenCodeZenBaseURL          = "https://opencode.ai/zen/v1"
	DefaultOpenCodeGoAnthropicBaseURL  = "https://opencode.ai/zen/go"
	DefaultOpenCodeZenAnthropicBaseURL = "https://opencode.ai/zen"
	openCodeProtocolRulesKey           = "protocol_rules"
	maxOpenCodeProtocolRules           = 64
	maxOpenCodeProtocolPatternLength   = 128
)

// DefaultOpenCodeGoModelIDs is the documented catalog fallback used before a
// live upstream model list is available.
func DefaultOpenCodeGoModelIDs() []string {
	return []string{
		"grok-4.7", "grok-4.6", "gpt-5.6-luna", "glm-5.3-flash", "glm-5.3",
		"glm-5.2", "glm-5.1", "kimi-k3", "kimi-k2.7-code", "kimi-k2.6",
		"longcat-2.0", "deepseek-v4-pro", "deepseek-v4-flash", "mimo-v2.5",
		"mimo-v2.5-pro", "minimax-m3", "minimax-m2.7", "minimax-m2.5",
		"muse-spark-1.3-contributor", "muse-spark-1.2-contributor", "qwen3.8-max",
		"qwen3.8-flash", "qwen3.7-max", "qwen3.7-plus", "qwen3.6-plus", "hy4-preview",
		"hy3", "omen-alpha",
	}
}

// IsOpenCodeGo reports whether the account uses the reserved OpenCode
// platform. It does not enable routing, persistence, or quota behavior.
func (a *Account) IsOpenCodeGo() bool {
	return a != nil && a.Platform == PlatformOpenCodeGo
}

// GetOpenCodeAccountMode returns the stored OpenCode mode. Existing OpenCode
// accounts without an explicit mode are treated as Go for compatibility.
func (a *Account) GetOpenCodeAccountMode() string {
	if !a.IsOpenCodeGo() {
		return ""
	}
	if a.Credentials != nil && strings.TrimSpace(openCodeStringValue(a.Credentials["account_mode"])) == AccountModeZen {
		return AccountModeZen
	}
	return AccountModeGo
}

func openCodeStringValue(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	return raw
}

// IsOpenCodeZen and IsOpenCodeGoPlan are pure account classification helpers.
// Actual routing and quota behavior is intentionally implemented in later
// OpenCode batches after the database contract is approved.
func (a *Account) IsOpenCodeZen() bool {
	return a.GetOpenCodeAccountMode() == AccountModeZen
}

func (a *Account) IsOpenCodeGoPlan() bool {
	return a.GetOpenCodeAccountMode() == AccountModeGo
}

func (a *Account) openCodeDefaultChatBaseURL() string {
	if a.IsOpenCodeZen() {
		return DefaultOpenCodeZenBaseURL
	}
	return DefaultOpenCodeGoBaseURL
}

func (a *Account) openCodeDefaultAnthropicBaseURL() string {
	if a.IsOpenCodeZen() {
		return DefaultOpenCodeZenAnthropicBaseURL
	}
	return DefaultOpenCodeGoAnthropicBaseURL
}

// GetAPIProtocol returns the configured protocol selector for OpenCode Go.
// Existing accounts default to adaptive routing; non-OpenCode accounts retain
// the historical Chat Completions fallback until their own protocol batch.
func (a *Account) GetAPIProtocol() string {
	if !a.IsOpenCodeGo() {
		return APIProtocolChatCompletions
	}
	configured := strings.TrimSpace(a.GetCredential("api_protocol"))
	switch configured {
	case APIProtocolChatCompletions, APIProtocolAnthropic, APIProtocolResponses, APIProtocolAdaptive:
		return configured
	default:
		return APIProtocolAdaptive
	}
}

// OpenCodeGoProtocolRule maps a model id or trailing-prefix pattern to a
// native upstream protocol. Rules are evaluated in order.
type OpenCodeGoProtocolRule struct {
	Pattern  string `json:"pattern"`
	Protocol string `json:"protocol"`
}

func DefaultOpenCodeGoProtocolRules() []OpenCodeGoProtocolRule {
	return []OpenCodeGoProtocolRule{
		{Pattern: "grok-*", Protocol: APIProtocolResponses},
		{Pattern: "gpt-*", Protocol: APIProtocolResponses},
		{Pattern: "muse-spark-*", Protocol: APIProtocolResponses},
		{Pattern: "minimax-*", Protocol: APIProtocolAnthropic},
		{Pattern: "qwen*", Protocol: APIProtocolAnthropic},
	}
}

func DefaultOpenCodeZenProtocolRules() []OpenCodeGoProtocolRule {
	return []OpenCodeGoProtocolRule{
		{Pattern: "grok-*", Protocol: APIProtocolResponses},
		{Pattern: "gpt-*", Protocol: APIProtocolResponses},
		{Pattern: "muse-spark-*", Protocol: APIProtocolResponses},
		{Pattern: "claude-*", Protocol: APIProtocolAnthropic},
		{Pattern: "qwen*", Protocol: APIProtocolAnthropic},
	}
}

func normalizeOpenCodeModelID(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range []string{"opencode-go/", "opencode_go/", "opencode/"} {
		model = strings.TrimPrefix(model, prefix)
	}
	return model
}

func normalizeOpenCodeProtocolPattern(pattern string) (string, error) {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	if len(pattern) > maxOpenCodeProtocolPatternLength {
		return "", fmt.Errorf("pattern is too long")
	}
	if strings.ContainsAny(pattern, " \t") {
		return "", fmt.Errorf("pattern must not contain whitespace")
	}
	if strings.Count(pattern, "*") > 1 || (strings.Contains(pattern, "*") && !strings.HasSuffix(pattern, "*")) {
		return "", fmt.Errorf("pattern may use a single trailing * wildcard")
	}
	return pattern, nil
}

func parseOpenCodeProtocolRules(raw any) ([]OpenCodeGoProtocolRule, error) {
	var items []any
	switch typed := raw.(type) {
	case []any:
		items = typed
	case []map[string]any:
		items = make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
	default:
		return nil, fmt.Errorf("protocol_rules must be an array")
	}
	if len(items) > maxOpenCodeProtocolRules {
		return nil, fmt.Errorf("protocol_rules supports at most %d entries", maxOpenCodeProtocolRules)
	}
	rules := make([]OpenCodeGoProtocolRule, 0, len(items))
	for i, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("protocol_rules[%d] must be an object", i)
		}
		pattern, _ := entry["pattern"].(string)
		protocol, _ := entry["protocol"].(string)
		var err error
		pattern, err = normalizeOpenCodeProtocolPattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("protocol_rules[%d]: %w", i, err)
		}
		protocol = strings.TrimSpace(protocol)
		switch protocol {
		case APIProtocolChatCompletions, APIProtocolAnthropic, APIProtocolResponses:
		default:
			return nil, fmt.Errorf("protocol_rules[%d]: unsupported protocol %q", i, protocol)
		}
		rules = append(rules, OpenCodeGoProtocolRule{Pattern: pattern, Protocol: protocol})
	}
	return rules, nil
}

func openCodeProtocolPatternMatches(pattern, model string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	model = normalizeOpenCodeModelID(model)
	if pattern == "" || model == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(model, strings.TrimSuffix(pattern, "*"))
	}
	return model == pattern
}

func matchOpenCodeProtocolRules(model string, rules []OpenCodeGoProtocolRule) string {
	for _, rule := range rules {
		if openCodeProtocolPatternMatches(rule.Pattern, model) {
			return rule.Protocol
		}
	}
	return APIProtocolChatCompletions
}

// ResolveOpenCodeGoUpstreamProtocol decides the native protocol without
// touching credentials outside the account's protocol_rules field.
func (a *Account) ResolveOpenCodeGoUpstreamProtocol(model string) string {
	if !a.IsOpenCodeGo() {
		return ""
	}
	switch configured := a.GetAPIProtocol(); configured {
	case APIProtocolChatCompletions, APIProtocolAnthropic, APIProtocolResponses:
		return configured
	}
	if raw, ok := a.Credentials[openCodeProtocolRulesKey]; ok && raw != nil {
		if rules, err := parseOpenCodeProtocolRules(raw); err == nil {
			return matchOpenCodeProtocolRules(model, rules)
		}
	}
	if a.IsOpenCodeZen() {
		return matchOpenCodeProtocolRules(model, DefaultOpenCodeZenProtocolRules())
	}
	return matchOpenCodeProtocolRules(model, DefaultOpenCodeGoProtocolRules())
}

// OpenCodeGoModelProtocol applies the built-in Go routing table without an
// account, for model catalog and connection-test callers.
func OpenCodeGoModelProtocol(model string) string {
	return matchOpenCodeProtocolRules(model, DefaultOpenCodeGoProtocolRules())
}

// OpenCodeGoQuotaURL normalizes both /zen/go and /zen/go/v1 bases to the
// provider's supported /v1/usage endpoint.
func OpenCodeGoQuotaURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = DefaultOpenCodeGoBaseURL
	}
	base = strings.TrimSuffix(base, "/v1")
	return base + "/v1/usage"
}

// ParseOpenCodeGoUsageResponse parses only the documented usage windows and
// discards all other upstream fields before they can enter a DTO or snapshot.
func ParseOpenCodeGoUsageResponse(body []byte) (*OpenCodeGoUsageData, error) {
	var payload struct {
		Usage map[string]json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode OpenCode Go usage: %w", err)
	}
	if payload.Usage == nil {
		return nil, fmt.Errorf("OpenCode Go usage field is missing")
	}
	window := func(key string) OpenCodeGoUsageWindow {
		raw, ok := payload.Usage[key]
		if !ok {
			return OpenCodeGoUsageWindow{Status: "unknown"}
		}
		var fields struct {
			Percent  json.RawMessage `json:"percent"`
			ResetsAt json.RawMessage `json:"resetsAt"`
		}
		if err := json.Unmarshal(raw, &fields); err != nil {
			return OpenCodeGoUsageWindow{Status: "failed"}
		}
		percent, ok := parseOpenCodePercent(fields.Percent)
		if !ok {
			return OpenCodeGoUsageWindow{Status: "failed"}
		}
		return OpenCodeGoUsageWindow{Status: OpenCodeGoUsageStatusOK, Percent: percent, ResetsAt: parseOpenCodeResetTime(fields.ResetsAt)}
	}
	return &OpenCodeGoUsageData{
		Rolling: window("rolling"),
		Weekly:  window("weekly"),
		Monthly: window("monthly"),
	}, nil
}

func parseOpenCodePercent(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		if number < 0 {
			number = 0
		}
		return number, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, false
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return 0, false
	}
	if number < 0 {
		number = 0
	}
	return number, true
}

func parseOpenCodeResetTime(raw json.RawMessage) time.Time {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(text)); err == nil {
			return parsed.UTC()
		}
	}
	var number float64
	if json.Unmarshal(raw, &number) == nil && number > 0 {
		if number >= 1e12 {
			number /= 1000
		}
		return time.Unix(int64(number), 0).UTC()
	}
	return time.Time{}
}

const (
	OpenCodeGoUsageStatusOK           = "ok"
	OpenCodeGoUsageStatusUnauthorized = "unauthorized"
	OpenCodeGoUsageStatusFailed       = "failed"
)

// OpenCodeGoUsageWindow is a sanitized view of one provider usage window.
// It contains no upstream payload, credential, or authorization field.
type OpenCodeGoUsageWindow struct {
	Status   string    `json:"status"`
	Percent  float64   `json:"percent"`
	ResetsAt time.Time `json:"resets_at"`
}

// OpenCodeGoUsageData contains only the three supported usage windows.
type OpenCodeGoUsageData struct {
	Rolling OpenCodeGoUsageWindow `json:"rolling"`
	Weekly  OpenCodeGoUsageWindow `json:"weekly"`
	Monthly OpenCodeGoUsageWindow `json:"monthly"`
}

// OpenCodeGoUsageSnapshot is a sanitized observation suitable for persistence
// after a later batch adds the repository contract.
type OpenCodeGoUsageSnapshot struct {
	Status        string               `json:"status"`
	Data          *OpenCodeGoUsageData `json:"data,omitempty"`
	FetchedAt     *time.Time           `json:"fetched_at,omitempty"`
	LastAttemptAt time.Time            `json:"last_attempt_at"`
	NextRefreshAt time.Time            `json:"next_refresh_at"`
	FailureCount  int                  `json:"failure_count,omitempty"`
	HTTPStatus    int                  `json:"http_status,omitempty"`
	LastError     string               `json:"last_error,omitempty"`
}

// OpenCodeGoUsageState is the read-only account-list/detail contract. It is
// not wired into a handler in this batch.
type OpenCodeGoUsageState struct {
	AccountID          int64                    `json:"account_id"`
	Eligible           bool                     `json:"eligible"`
	AutoRefreshEnabled bool                     `json:"auto_refresh_enabled"`
	Snapshot           *OpenCodeGoUsageSnapshot `json:"snapshot,omitempty"`
}
