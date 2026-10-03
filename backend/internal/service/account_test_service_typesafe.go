package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"anlapi/internal/pkg/typesafe"
	"github.com/gin-gonic/gin"
)

const (
	typeSafeTestDefaultState = "ANLAPI TypeSafe connection test"
	typeSafeTestQuestionID   = "connection_test"
)

// testTypeSafeAccountConnection probes the native, non-streaming System One API.
// TypeSafe keys must never fall through to the Claude /v1/messages probe.
func (s *AccountTestService) testTypeSafeAccountConnection(c *gin.Context, account *Account, prompt string) error {
	ctx := c.Request.Context()
	if account.Type != AccountTypeAPIKey {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}
	apiKey := account.GetTypeSafeAPIKey()
	if apiKey == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}
	baseURL, err := s.validateUpstreamBaseURL(account.GetTypeSafeBaseURL())
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
	}
	state := strings.TrimSpace(prompt)
	if state == "" {
		state = typeSafeTestDefaultState
	}
	payload, err := json.Marshal(map[string]any{
		"model": typesafe.JevLatestModel,
		"state": state,
		"questions": map[string]any{
			typeSafeTestQuestionID: map[string]any{
				"type":         "noul",
				"instructions": "Is this text a connection test?",
			},
		},
	})
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create TypeSafe test payload")
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()
	s.sendEvent(c, TestEvent{Type: "test_start", Model: typesafe.JevLatestModel})

	req, err := typesafe.NewSystemOneRequest(ctx, baseURL, apiKey, payload)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create TypeSafe request")
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("TypeSafe request failed: %s", sanitizeUpstreamErrorMessage(err.Error())))
	}
	if resp == nil {
		return s.sendErrorAndEnd(c, "TypeSafe request failed: empty response")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && s.accountRepo != nil {
			_ = s.accountRepo.SetError(ctx, account.ID, fmt.Sprintf("TypeSafe API rejected credentials (%d)", resp.StatusCode))
		}
		return s.sendErrorAndEnd(c, fmt.Sprintf("TypeSafe API returned %d", resp.StatusCode))
	}

	decoded, err := typesafe.DecodeSystemOneResponse(resp.Body)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid TypeSafe response: %s", err.Error()))
	}
	s.sendEvent(c, TestEvent{Type: "content", Text: string(decoded.Body)})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
