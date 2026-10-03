package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"anlapi/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestResolveOpenAICompactFallbackModel_PrefersAccountMapping(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{OpenAICompactModel: "gpt-5.4"}}}
	account := &Account{Credentials: map[string]any{
		"compact_model_mapping": map[string]any{"gpt-5.4": "gpt-5.4-account"},
	}}

	require.Equal(t, "gpt-5.4-account", svc.resolveOpenAICompactFallbackModel(account, "gpt-5.4"))
	require.Equal(t, "gpt-5.4", svc.resolveOpenAICompactFallbackModel(&Account{}, "gpt-5.4"))
}

func TestPrepareOpenAICompactFallbackRetry_AllowsOneSameAccountRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{OpenAICompactModel: "gpt-5.4"}}}
	body := []byte(`{"model":"gpt-5.5","input":[]}`)
	upstreamBody := []byte(`{"error":{"code":"model_not_found","message":"model unavailable"}}`)

	retryBody, fallback, retry := svc.prepareOpenAICompactFallbackRetry(c, &Account{}, "gpt-5.5", body, http.StatusBadRequest, "model unavailable", upstreamBody, false)
	require.True(t, retry)
	require.Equal(t, "gpt-5.4", fallback)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(retryBody, "model").String())

	_, _, retry = svc.prepareOpenAICompactFallbackRetry(c, &Account{}, "gpt-5.5", retryBody, http.StatusBadRequest, "model unavailable", upstreamBody, true)
	require.False(t, retry)
}

func TestPrepareOpenAICompactFallbackRetry_RecognizesNativeTrigger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	MarkOpenAINativeCompactionV2(c)
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{OpenAICompactModel: "gpt-5.4"}}}
	body := []byte(`{"model":"gpt-5.5","input":[{"type":"compaction_trigger"}]}`)

	_, fallback, retry := svc.prepareOpenAICompactFallbackRetry(c, &Account{}, "gpt-5.5", body, http.StatusBadRequest, "model unavailable", []byte(`{"error":{"code":"unsupported_model"}}`), false)
	require.True(t, retry)
	require.Equal(t, "gpt-5.4", fallback)
}

func TestHandleSSEToJSON_CompactFailureReturnsFallbackSignal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	body := []byte("data: {\"type\":\"response.failed\",\"error\":{\"code\":\"model_not_found\",\"message\":\"model unavailable\"}}\n\ndata: [DONE]\n")

	result, err := svc.handleSSEToJSON(resp, c, &Account{ID: 1, Platform: PlatformOpenAI}, body, "gpt-5.5", "gpt-5.5")
	require.Nil(t, result)
	var signal *openAICompactFallbackSignal
	require.True(t, errors.As(err, &signal))
	require.NotEmpty(t, signal.payload)
	require.False(t, c.Writer.Written())
}

func TestHandlePassthroughSSEToJSON_CompactFailureReturnsFallbackSignal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	body := []byte("data: {\"type\":\"response.failed\",\"error\":{\"code\":\"model_not_found\",\"message\":\"model unavailable\"}}\n\ndata: [DONE]\n")

	result, err := svc.handlePassthroughSSEToJSON(resp, c, body, "gpt-5.5", "gpt-5.5")
	require.Nil(t, result)
	var signal *openAICompactFallbackSignal
	require.True(t, errors.As(err, &signal))
	require.NotEmpty(t, signal.payload)
	require.False(t, c.Writer.Written())
}
