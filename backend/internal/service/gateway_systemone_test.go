package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"anlapi/internal/config"
	"anlapi/internal/pkg/tlsfingerprint"
	"anlapi/internal/pkg/typesafe"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemOneHTTPUpstream struct {
	do func(*http.Request) (*http.Response, error)
}

func (u *systemOneHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return u.do(req)
}

func (u *systemOneHTTPUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.do(req)
}

func newSystemOneTestService(upstream HTTPUpstream) *GatewayService {
	return &GatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{AllowInsecureHTTP: true}}},
		httpUpstream: upstream,
	}
}

func newSystemOneTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", nil)
	return c
}

func TestForwardSystemOneForwardsNativeProtocolAndUsage(t *testing.T) {
	requestBody := []byte(`{"model":"jev-latest","state":{"text":"sample"},"questions":{"q":{"type":"choice","instructions":"Pick","criteria":{"a":"A","b":"B"}}}}`)
	responseBody := []byte(`{"model":"jev-1.13.0","answers":{"q":{"type":"choice","choice":"a"}},"usage":{"input_tokens":123,"output_tokens":7},"provider_extension":{"kept":true}}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, typesafe.SystemOnePath, r.URL.Path)
		require.Equal(t, "Bearer ts-secret", r.Header.Get("Authorization"))
		got, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, requestBody, got)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("x-request-id", "req-jev")
		_, err = w.Write(responseBody)
		require.NoError(t, err)
	}))
	defer server.Close()

	account := &Account{ID: 7, Platform: PlatformTypeSafe, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": server.URL, "api_key": "ts-secret"}}
	result, err := newSystemOneTestService(&systemOneHTTPUpstream{do: server.Client().Do}).ForwardSystemOne(context.Background(), newSystemOneTestContext(), account, requestBody)
	require.NoError(t, err)
	require.Equal(t, responseBody, result.Body)
	require.Equal(t, "req-jev", result.RequestID)
	require.Equal(t, typesafe.JevLatestModel, result.Model)
	require.Equal(t, typesafe.JevLatestModel, result.UpstreamModel)
	require.Equal(t, 123, result.Usage.InputTokens)
	require.Equal(t, "application/json; charset=utf-8", result.ContentType)
}

func TestForwardSystemOneErrorPolicyAndRedaction(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusBadGateway} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			upstream := &systemOneHTTPUpstream{do: func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("private body ts-secret"))}, nil
			}}
			account := &Account{ID: 8, Platform: PlatformTypeSafe, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "http://typesafe.test", "api_key": "ts-secret"}}
			_, err := newSystemOneTestService(upstream).ForwardSystemOne(context.Background(), newSystemOneTestContext(), account, []byte(`{}`))
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private")
			require.NotContains(t, err.Error(), "ts-secret")
			if IsSystemOneRequestErrorStatus(status) {
				var requestErr *SystemOneUpstreamError
				require.ErrorAs(t, err, &requestErr)
			} else {
				var failoverErr *UpstreamFailoverError
				require.ErrorAs(t, err, &failoverErr)
			}
		})
	}
}

func TestForwardSystemOneTransportFailureIsRetryable(t *testing.T) {
	account := &Account{ID: 9, Platform: PlatformTypeSafe, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "http://typesafe.test", "api_key": "ts-secret"}}
	upstream := &systemOneHTTPUpstream{do: func(*http.Request) (*http.Response, error) { return nil, errors.New("dial failed for ts-secret") }}
	_, err := newSystemOneTestService(upstream).ForwardSystemOne(context.Background(), newSystemOneTestContext(), account, []byte(`{}`))
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.NotContains(t, err.Error(), "ts-secret")
}

func TestForwardSystemOneRejectsOversizedResponse(t *testing.T) {
	oversized := `{"pad":"` + strings.Repeat("a", typesafe.MaxSystemOneResponseBytes) + `"}`
	upstream := &systemOneHTTPUpstream{do: func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(oversized))}, nil
	}}
	account := &Account{ID: 10, Platform: PlatformTypeSafe, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "http://typesafe.test", "api_key": "ts-secret"}}
	_, err := newSystemOneTestService(upstream).ForwardSystemOne(context.Background(), newSystemOneTestContext(), account, []byte(`{}`))
	require.ErrorIs(t, err, typesafe.ErrSystemOneResponseTooLarge)
}

func TestSystemOneResponseContentTypeIsJSONOnly(t *testing.T) {
	require.Equal(t, "application/json; charset=utf-8", systemOneResponseContentType("application/json; charset=utf-8"))
	require.Equal(t, "application/problem+json", systemOneResponseContentType("application/problem+json"))
	require.Equal(t, "application/json", systemOneResponseContentType("text/html; charset=utf-8"))
	require.Equal(t, "application/json", systemOneResponseContentType("not a media type;;"))
}
