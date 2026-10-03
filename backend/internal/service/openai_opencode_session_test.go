package service

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApplyOpenCodeSessionHeaderUsesStableSignals(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	account := &Account{
		Platform:    PlatformOpenCodeGo,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "redacted"},
	}
	headers := make(http.Header)
	applyOpenCodeSessionHeader(c, account, DefaultOpenCodeGoBaseURL+"/chat/completions", headers, []byte(`{"prompt_cache_key":"turn-42"}`))
	if got := headers.Get(openCodeSessionHeader); got != "turn-42" {
		t.Fatalf("expected prompt_cache_key to become session header, got %q", got)
	}

	c.Request.Header.Set(openCodeSessionIDHeader, "header-session")
	headers = make(http.Header)
	applyOpenCodeSessionHeader(c, account, DefaultOpenCodeGoBaseURL+"/chat/completions", headers, []byte(`{"prompt_cache_key":"body-session"}`))
	if got := headers.Get(openCodeSessionHeader); got != "header-session" {
		t.Fatalf("expected stable client header to take precedence, got %q", got)
	}
}

func TestApplyOpenCodeSessionHeaderDoesNotTouchOtherPlatforms(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	headers := make(http.Header)
	applyOpenCodeSessionHeader(c, &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, DefaultOpenCodeGoBaseURL+"/chat/completions", headers, []byte(`{"prompt_cache_key":"no-cross-platform"}`))
	if got := headers.Get(openCodeSessionHeader); got != "" {
		t.Fatalf("non-OpenCode account received a session header: %q", got)
	}
}
