package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"anlapi/internal/pkg/antigravity"
	kiropkg "anlapi/internal/pkg/kiro"
	"github.com/stretchr/testify/require"
)

type retryKiroOAuthClient struct {
	socialCalls int
	idcCalls    int
}

func (c *retryKiroOAuthClient) createSocialToken(context.Context, string, string, string, string) (*kiropkg.TokenData, error) {
	c.socialCalls++
	if c.socialCalls == 1 {
		return nil, errors.New("temporary social token exchange failure")
	}
	return &kiropkg.TokenData{AccessToken: "test-social-access", RefreshToken: "test-social-refresh", AuthMethod: "social"}, nil
}

func (c *retryKiroOAuthClient) exchangeIDCAuthCode(context.Context, string, string, string, string, string, string, string, string) (*kiropkg.TokenData, error) {
	c.idcCalls++
	if c.idcCalls == 1 {
		return nil, errors.New("temporary IDC token exchange failure")
	}
	return &kiropkg.TokenData{AccessToken: "test-idc-access", RefreshToken: "test-idc-refresh", AuthMethod: "idc"}, nil
}

func TestKiroOAuthService_ExchangeCodeCanRetryAfterProviderFailure(t *testing.T) {
	tests := []struct {
		name       string
		authType   string
		provider   string
		wantAccess string
		wantError  string
	}{
		{name: "social", authType: "social", provider: "Google", wantAccess: "test-social-access", wantError: "temporary social token exchange failure"},
		{name: "idc", authType: "idc", provider: "AWS", wantAccess: "test-idc-access", wantError: "temporary IDC token exchange failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &retryKiroOAuthClient{}
			svc := NewKiroOAuthService(nil)
			svc.oauthClient = client
			const sessionID = "kiro-provider-retry-session"
			svc.sessionStore.Set(sessionID, &kiropkg.AuthSession{
				State:        "expected-state",
				CodeVerifier: "test-verifier",
				CreatedAt:    time.Now(),
				AuthType:     tt.authType,
				Provider:     tt.provider,
				RedirectURI:  kiroSocialRedirectURI,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Region:       "us-east-1",
				StartURL:     "https://example.test/start",
			})
			input := &KiroExchangeCodeInput{SessionID: sessionID, State: "expected-state", Code: "test-code"}

			_, err := svc.ExchangeCode(context.Background(), input)
			require.ErrorContains(t, err, tt.wantError)
			_, ok := svc.sessionStore.Get(sessionID)
			require.True(t, ok, "provider failure must preserve the pending session for retry")

			result, err := svc.ExchangeCode(context.Background(), input)
			require.NoError(t, err)
			require.Equal(t, tt.wantAccess, result.AccessToken)
			_, ok = svc.sessionStore.Get(sessionID)
			require.False(t, ok, "a successful exchange consumes the pending session")
			require.Equal(t, 2, client.socialCalls+client.idcCalls, "the selected flow must be called once for each attempt")
		})
	}
}

type retryAntigravityOAuthClient struct {
	exchangeCalls int
}

func (c *retryAntigravityOAuthClient) ExchangeCode(context.Context, string, string) (*antigravity.TokenResponse, error) {
	c.exchangeCalls++
	if c.exchangeCalls == 1 {
		return nil, errors.New("temporary token exchange failure")
	}
	return &antigravity.TokenResponse{
		AccessToken:  "test-antigravity-access",
		RefreshToken: "test-antigravity-refresh",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}, nil
}

func (*retryAntigravityOAuthClient) GetUserInfo(context.Context, string) (*antigravity.UserInfo, error) {
	return &antigravity.UserInfo{Email: "oauth-test@example.invalid"}, nil
}

func (*retryAntigravityOAuthClient) LoadCodeAssist(context.Context, string) (*antigravity.LoadCodeAssistResponse, map[string]any, error) {
	return &antigravity.LoadCodeAssistResponse{CloudAICompanionProject: "test-project"}, nil, nil
}

func (*retryAntigravityOAuthClient) OnboardUser(context.Context, string, string) (string, error) {
	return "", errors.New("unexpected project onboarding call")
}

func TestAntigravityOAuthService_ExchangeCodeCanRetryAfterProviderFailure(t *testing.T) {
	client := &retryAntigravityOAuthClient{}
	svc := NewAntigravityOAuthService(nil)
	svc.newOAuthClient = func(string) (antigravityOAuthClient, error) { return client, nil }
	svc.privacySetter = func(context.Context, string, string, string) string { return "test-disabled" }
	const sessionID = "antigravity-provider-retry-session"
	svc.sessionStore.Set(sessionID, &antigravity.OAuthSession{
		State:        "expected-state",
		CodeVerifier: "test-verifier",
		CreatedAt:    time.Now(),
	})
	input := &AntigravityExchangeCodeInput{SessionID: sessionID, State: "expected-state", Code: "test-code"}

	_, err := svc.ExchangeCode(context.Background(), input)
	require.ErrorContains(t, err, "temporary token exchange failure")
	_, ok := svc.sessionStore.Get(sessionID)
	require.True(t, ok, "provider failure must preserve the pending session for retry")

	result, err := svc.ExchangeCode(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "test-antigravity-access", result.AccessToken)
	require.Equal(t, "oauth-test@example.invalid", result.Email)
	require.Equal(t, "test-project", result.ProjectID)
	require.Equal(t, "test-disabled", result.PrivacyMode)
	_, ok = svc.sessionStore.Get(sessionID)
	require.False(t, ok, "a successful exchange consumes the pending session")
	require.Equal(t, 2, client.exchangeCalls)
}
