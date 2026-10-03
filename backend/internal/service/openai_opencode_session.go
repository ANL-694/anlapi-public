package service

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const openCodeSessionHeader = openCodeNativeSessionHeader

// applyOpenCodeSessionHeader attaches the stable OpenCode conversation header
// without forwarding arbitrary client credentials or request identifiers.
func applyOpenCodeSessionHeader(c *gin.Context, account *Account, targetURL string, headers http.Header, body []byte) {
	if account == nil || !account.IsOpenCodeGo() || account.Type != AccountTypeAPIKey || headers == nil {
		return
	}
	if !shouldSendOpenCodeSessionHeader(account, targetURL) {
		return
	}

	sessionID := resolveOpenCodeSessionID(c, headers, body)
	if sessionID == "" {
		return
	}
	headers.Set(openCodeSessionHeader, sessionID)
}

func shouldSendOpenCodeSessionHeader(account *Account, targetURL string) bool {
	if account != nil && account.IsOpenCodeGoPlan() {
		return true
	}
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	return err == nil && strings.EqualFold(parsed.Scheme, "https") &&
		strings.EqualFold(parsed.Hostname(), "opencode.ai")
}

func resolveOpenCodeSessionID(c *gin.Context, headers http.Header, body []byte) string {
	if c != nil {
		if sessionID := explicitOpenAIHeaderSessionID(c); sessionID != "" {
			return sessionID
		}
	}
	if sessionID := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); sessionID != "" {
		return sessionID
	}
	if userID := strings.TrimSpace(gjson.GetBytes(body, "metadata.user_id").String()); userID != "" {
		return userID
	}
	if sessionID := existingOpenCodeSessionHeader(headers); sessionID != "" {
		return sessionID
	}
	return uuid.NewString()
}

func existingOpenCodeSessionHeader(headers http.Header) string {
	if headers == nil {
		return ""
	}
	return strings.TrimSpace(headers.Get(openCodeSessionHeader))
}
