package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOpenCodeGoAccountContractDefaultsSafely(t *testing.T) {
	t.Parallel()

	legacy := &Account{Platform: PlatformOpenCodeGo}
	if !legacy.IsOpenCodeGo() {
		t.Fatal("expected OpenCode platform to be recognized")
	}
	if got := legacy.GetOpenCodeAccountMode(); got != AccountModeGo {
		t.Fatalf("expected missing mode to default to %q, got %q", AccountModeGo, got)
	}
	if !legacy.IsOpenCodeGoPlan() || legacy.IsOpenCodeZen() {
		t.Fatal("expected legacy OpenCode account to be classified as Go")
	}

	zen := &Account{
		Platform:    PlatformOpenCodeGo,
		Credentials: map[string]any{"account_mode": AccountModeZen},
	}
	if got := zen.GetOpenCodeAccountMode(); got != AccountModeZen {
		t.Fatalf("expected explicit Zen mode, got %q", got)
	}
	if !zen.IsOpenCodeZen() || zen.IsOpenCodeGoPlan() {
		t.Fatal("expected explicit Zen account to be classified as Zen")
	}

	other := &Account{Platform: PlatformOpenAI, Credentials: map[string]any{"account_mode": AccountModeZen}}
	if got := other.GetOpenCodeAccountMode(); got != "" {
		t.Fatalf("expected non-OpenCode account to have no OpenCode mode, got %q", got)
	}
}

func TestOpenCodeGoOpenAIGatewayAccountContract(t *testing.T) {
	t.Parallel()

	goAccount := &Account{
		Platform:    PlatformOpenCodeGo,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	if got := goAccount.GetOpenAIBaseURL(); got != DefaultOpenCodeGoBaseURL {
		t.Fatalf("expected Go default base URL %q, got %q", DefaultOpenCodeGoBaseURL, got)
	}
	if got := goAccount.GetOpenAIApiKey(); got != "sk-test" {
		t.Fatalf("expected OpenCode API key to be available to the gateway, got %q", got)
	}
	if !goAccount.IsOpenAICompatible() {
		t.Fatal("expected OpenCode account to be OpenAI-compatible")
	}
	if !goAccount.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions) {
		t.Fatal("expected raw Chat Completions capability")
	}
	if goAccount.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses) {
		t.Fatal("the standalone Responses endpoint must remain gated")
	}
	goAccount.Credentials["protocol_rules"] = []any{map[string]any{"pattern": "gpt-*", "protocol": APIProtocolResponses}}
	if !shouldUseResponsesAPIForAccountModel(goAccount, "gpt-5.6-luna", "") {
		t.Fatal("a model explicitly routed to Responses should use the existing Responses adapter")
	}
	if shouldUseResponsesAPIForAccountModel(goAccount, "glm-5.3", "") {
		t.Fatal("a model without a Responses rule must remain on native Chat")
	}

	zenAccount := &Account{
		Platform:    PlatformOpenCodeGo,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"account_mode": AccountModeZen, "base_url": "https://proxy.example/zen/v1"},
	}
	if got := zenAccount.GetOpenAIBaseURL(); got != "https://proxy.example/zen/v1" {
		t.Fatalf("expected explicit base URL to win, got %q", got)
	}
	zenAccount.Credentials["base_url"] = nil
	if got := zenAccount.GetOpenAIBaseURL(); got != DefaultOpenCodeZenBaseURL {
		t.Fatalf("expected Zen default base URL %q, got %q", DefaultOpenCodeZenBaseURL, got)
	}
}

func TestOpenCodeGoUsageContractIsSanitizedAndStable(t *testing.T) {
	t.Parallel()

	reset := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	state := OpenCodeGoUsageState{
		AccountID:          42,
		Eligible:           true,
		AutoRefreshEnabled: false,
		Snapshot: &OpenCodeGoUsageSnapshot{
			Status:        OpenCodeGoUsageStatusOK,
			Data:          &OpenCodeGoUsageData{Rolling: OpenCodeGoUsageWindow{Status: "ok", Percent: 12.5, ResetsAt: reset}},
			LastAttemptAt: reset,
			NextRefreshAt: reset.Add(time.Hour),
		},
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal usage state: %v", err)
	}
	jsonText := string(raw)
	for _, forbidden := range []string{"credentials", "authorization", "api_key", "cookie", "access_token"} {
		if containsFold(jsonText, forbidden) {
			t.Fatalf("sanitized DTO contains forbidden field %q: %s", forbidden, jsonText)
		}
	}
	if !containsFold(jsonText, `"rolling"`) || !containsFold(jsonText, `"percent":12.5`) {
		t.Fatalf("usage windows were not serialized: %s", jsonText)
	}
}

func TestOpenCodeGoProtocolRoutingIsSafeAndOrdered(t *testing.T) {
	t.Parallel()

	account := &Account{Platform: PlatformOpenCodeGo, Credentials: map[string]any{}}
	if got := account.ResolveOpenCodeGoUpstreamProtocol("grok-4.6"); got != APIProtocolResponses {
		t.Fatalf("expected Grok to use responses, got %q", got)
	}
	if got := account.ResolveOpenCodeGoUpstreamProtocol("glm-5.3"); got != APIProtocolChatCompletions {
		t.Fatalf("expected unknown model to use chat completions, got %q", got)
	}

	account.Credentials["protocol_rules"] = []any{
		map[string]any{"pattern": "gpt-*", "protocol": APIProtocolChatCompletions},
		map[string]any{"pattern": "gpt-5.6-luna", "protocol": APIProtocolResponses},
	}
	if got := account.ResolveOpenCodeGoUpstreamProtocol("gpt-5.6-luna"); got != APIProtocolChatCompletions {
		t.Fatalf("expected first matching rule to win, got %q", got)
	}
}

func TestOpenCodeGoProtocolRulesRejectUnsafePatterns(t *testing.T) {
	t.Parallel()

	_, err := parseOpenCodeProtocolRules([]any{
		map[string]any{"pattern": "*grok", "protocol": APIProtocolResponses},
	})
	if err == nil {
		t.Fatal("expected non-trailing wildcard to be rejected")
	}
	_, err = parseOpenCodeProtocolRules([]any{
		map[string]any{"pattern": "grok-*", "protocol": APIProtocolAdaptive},
	})
	if err == nil {
		t.Fatal("expected adaptive to be rejected as a native upstream protocol")
	}
}

func TestOpenCodeGoUsageParserSanitizesWindows(t *testing.T) {
	t.Parallel()

	data, err := ParseOpenCodeGoUsageResponse([]byte(`{"usage":{"rolling":{"percent":"12.5","resetsAt":"2026-09-07T12:00:00Z"},"weekly":{"percent":40},"monthly":{"percent":22.2,"resetsAt":1790812800000},"secret":"discard"}}`))
	if err != nil {
		t.Fatalf("parse usage: %v", err)
	}
	if data.Rolling.Status != OpenCodeGoUsageStatusOK || data.Rolling.Percent != 12.5 {
		t.Fatalf("unexpected rolling window: %#v", data.Rolling)
	}
	if data.Weekly.Status != OpenCodeGoUsageStatusOK || data.Monthly.Status != OpenCodeGoUsageStatusOK {
		t.Fatalf("expected all supported windows to be usable: %#v", data)
	}
	if !data.Rolling.ResetsAt.Equal(time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected rolling reset: %s", data.Rolling.ResetsAt)
	}
	if _, err := ParseOpenCodeGoUsageResponse([]byte(`{"ok":true}`)); err == nil {
		t.Fatal("expected missing usage field to fail")
	}
}

func TestOpenCodeGoQuotaURLNormalizesBase(t *testing.T) {
	t.Parallel()
	if got := OpenCodeGoQuotaURL(DefaultOpenCodeGoBaseURL + "/"); got != "https://opencode.ai/zen/go/v1/usage" {
		t.Fatalf("unexpected quota URL: %s", got)
	}
	if got := OpenCodeGoQuotaURL(DefaultOpenCodeZenBaseURL); got != "https://opencode.ai/zen/v1/usage" {
		t.Fatalf("unexpected Zen quota URL: %s", got)
	}
}

func containsFold(value, fragment string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(fragment))
}
