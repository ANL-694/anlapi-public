package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	infraerrors "anlapi/internal/pkg/errors"
	"anlapi/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

type grokOAuthStateClientStub struct {
	exchangeCalled  int32
	lastRedirectURI string
	exchangeErr     error
}

func (s *grokOAuthStateClientStub) ExchangeCode(_ context.Context, _, _, redirectURI, _, _ string) (*xai.TokenResponse, error) {
	atomic.AddInt32(&s.exchangeCalled, 1)
	s.lastRedirectURI = redirectURI
	if s.exchangeErr != nil {
		return nil, s.exchangeErr
	}
	return &xai.TokenResponse{AccessToken: "at"}, nil
}

func (s *grokOAuthStateClientStub) RefreshToken(context.Context, string, string, string) (*xai.TokenResponse, error) {
	return nil, nil
}

func (s *grokOAuthStateClientStub) ConvertSSOToBuild(context.Context, string, string) (*xai.TokenResponse, error) {
	return nil, nil
}

func TestGrokOAuthService_ExchangeCode_RedirectURIMismatch(t *testing.T) {
	client := &grokOAuthStateClientStub{}
	svc := NewGrokOAuthService(nil, client)
	defer svc.Stop()

	svc.sessionStore.Set("sid", &xai.OAuthSession{
		State:        "expected-state",
		CodeVerifier: "verifier",
		RedirectURI:  xai.DefaultRedirectURI,
		CreatedAt:    time.Now(),
	})

	_, err := svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID:   "sid",
		Code:        "auth-code",
		State:       "expected-state",
		RedirectURI: "http://localhost:9999/callback",
	})
	require.Equal(t, "GROK_OAUTH_REDIRECT_URI_MISMATCH", infraerrors.Reason(err))
	require.Equal(t, int32(0), atomic.LoadInt32(&client.exchangeCalled))
	_, ok := svc.sessionStore.Get("sid")
	require.True(t, ok, "a redirect mismatch must leave the OAuth session available for a corrected retry")
}

func TestGrokOAuthService_ExchangeCode_RedirectURIMatch(t *testing.T) {
	client := &grokOAuthStateClientStub{}
	svc := NewGrokOAuthService(nil, client)
	defer svc.Stop()

	redirectURI := "http://localhost:1456/callback"
	svc.sessionStore.Set("sid", &xai.OAuthSession{
		State:        "expected-state",
		CodeVerifier: "verifier",
		RedirectURI:  redirectURI,
		ClientID:     xai.DefaultClientID,
		CreatedAt:    time.Now(),
	})

	info, err := svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID:   "sid",
		Code:        "auth-code",
		State:       "expected-state",
		RedirectURI: redirectURI,
	})
	require.NoError(t, err)
	require.Equal(t, "at", info.AccessToken)
	require.Equal(t, redirectURI, client.lastRedirectURI)
	require.Equal(t, int32(1), atomic.LoadInt32(&client.exchangeCalled))
	_, ok := svc.sessionStore.Get("sid")
	require.False(t, ok)
}

func TestGrokOAuthService_ExchangeCode_FailedExchangeKeepsSessionForRetry(t *testing.T) {
	client := &grokOAuthStateClientStub{exchangeErr: errors.New("provider temporarily unavailable")}
	svc := NewGrokOAuthService(nil, client)
	defer svc.Stop()

	svc.sessionStore.Set("sid", &xai.OAuthSession{
		State:        "expected-state",
		CodeVerifier: "verifier",
		RedirectURI:  xai.DefaultRedirectURI,
		CreatedAt:    time.Now(),
	})

	_, err := svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID: "sid",
		Code:      "auth-code",
		State:     "expected-state",
	})
	require.EqualError(t, err, "provider temporarily unavailable")
	_, ok := svc.sessionStore.Get("sid")
	require.True(t, ok, "a failed provider exchange must leave the session retryable")

	client.exchangeErr = nil
	info, err := svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID: "sid",
		Code:      "auth-code-retry",
		State:     "expected-state",
	})
	require.NoError(t, err)
	require.Equal(t, "at", info.AccessToken)
	require.Equal(t, int32(2), atomic.LoadInt32(&client.exchangeCalled))
	_, ok = svc.sessionStore.Get("sid")
	require.False(t, ok)
}

func TestGrokOAuthService_ExchangeCode_InvalidStateKeepsSession(t *testing.T) {
	client := &grokOAuthStateClientStub{}
	svc := NewGrokOAuthService(nil, client)
	defer svc.Stop()

	svc.sessionStore.Set("sid", &xai.OAuthSession{
		State:        "expected-state",
		CodeVerifier: "verifier",
		RedirectURI:  xai.DefaultRedirectURI,
		CreatedAt:    time.Now(),
	})

	_, err := svc.ExchangeCode(context.Background(), &GrokExchangeCodeInput{
		SessionID: "sid",
		Code:      "auth-code",
		State:     "wrong-state",
	})
	require.Equal(t, "GROK_OAUTH_INVALID_STATE", infraerrors.Reason(err))
	require.Equal(t, int32(0), atomic.LoadInt32(&client.exchangeCalled))
	_, ok := svc.sessionStore.Get("sid")
	require.True(t, ok, "an invalid state must not consume the legitimate OAuth session")
}
