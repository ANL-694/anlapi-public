package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"anlapi/internal/config"
	"github.com/gin-gonic/gin"
)

func TestOpenCodeChatAnthropicAdapterConvertsAndKeepsUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"minimax-m3\",\"usage\":{\"input_tokens\":2,\"output_tokens\":0}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":3}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")),
	}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"minimax-m3","messages":[{"role":"user","content":"hi"}],"stream":false}`)))
	account := &Account{
		ID:       701,
		Platform: PlatformOpenCodeGo,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":       "redacted",
			"base_url":      "https://opencode.example",
			"model_mapping": map[string]any{"client-model": "minimax-m3"},
		},
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	result, err := svc.ForwardAsChatCompletions(c.Request.Context(), c, account, []byte(`{"model":"client-model","messages":[{"role":"user","content":"hi"}],"stream":false}`), "", "")
	if err != nil {
		t.Fatalf("OpenCode native adapter failed: %v", err)
	}
	if result == nil || result.Usage.InputTokens != 2 || result.Usage.OutputTokens != 3 {
		t.Fatalf("unexpected usage result: %#v", result)
	}
	if result.Model != "client-model" || result.UpstreamModel != "minimax-m3" {
		t.Fatalf("expected mapped model to drive native protocol selection, got %#v", result)
	}
	if upstream.lastReq == nil || upstream.lastReq.URL.String() != "https://opencode.example/v1/messages?beta=true" {
		t.Fatalf("unexpected native target URL: %v", upstream.lastReq)
	}
	if got := upstream.lastReq.Header.Get(openCodeSessionHeader); got == "" {
		t.Fatal("expected OpenCode session header on native request")
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"object":"chat.completion"`)) {
		t.Fatalf("expected Chat Completions response, got %s", recorder.Body.String())
	}
}

func TestOpenCodeResponsesUpstreamRequestUsesNativeEndpointAndSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"gpt-5.6-luna","messages":[]}`)))
	account := &Account{
		ID:       702,
		Platform: PlatformOpenCodeGo,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":        "redacted",
			"base_url":       "https://opencode.example/zen/go/v1",
			"protocol_rules": []any{map[string]any{"pattern": "gpt-*", "protocol": APIProtocolResponses}},
		},
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	req, err := svc.buildUpstreamRequest(c.Request.Context(), c, account, []byte(`{"model":"gpt-5.6-luna","input":[]}`), "redacted", true, "", false)
	if err != nil {
		t.Fatalf("build OpenCode Responses request: %v", err)
	}
	if got := req.URL.String(); got != "https://opencode.example/zen/go/v1/responses" {
		t.Fatalf("unexpected OpenCode Responses URL: %s", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer redacted" {
		t.Fatalf("unexpected authorization header: %q", got)
	}
	if got := req.Header.Get(openCodeSessionHeader); got == "" {
		t.Fatal("expected OpenCode session header on Responses request")
	}
}
