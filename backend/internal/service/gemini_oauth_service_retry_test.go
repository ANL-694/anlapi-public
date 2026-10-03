//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"anlapi/internal/config"
	"anlapi/internal/pkg/geminicli"
	"github.com/stretchr/testify/require"
)

func TestGeminiOAuthService_ExchangeCode_RetryAfterProviderFailureKeepsSession(t *testing.T) {
	t.Parallel()

	calls := 0
	client := &mockGeminiOAuthClient{
		exchangeCodeFunc: func(context.Context, string, string, string, string, string) (*geminicli.TokenResponse, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("temporary provider failure")
			}
			return &geminicli.TokenResponse{
				AccessToken:  "gemini-retry-access-token",
				RefreshToken: "gemini-retry-refresh-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			}, nil
		},
	}

	svc := NewGeminiOAuthService(nil, client, nil, nil, &config.Config{
		Gemini: config.GeminiConfig{
			OAuth: config.GeminiOAuthConfig{
				ClientID:     "test-ai-studio-client",
				ClientSecret: "test-ai-studio-secret",
			},
		},
	})
	defer svc.Stop()

	const sessionID = "gemini-retry-session"
	svc.sessionStore.Set(sessionID, &geminicli.OAuthSession{
		State:        "state-1",
		CodeVerifier: "verifier-1",
		OAuthType:    "ai_studio",
		CreatedAt:    time.Now(),
	})

	input := &GeminiExchangeCodeInput{SessionID: sessionID, State: "state-1", Code: "retry-code"}
	_, err := svc.ExchangeCode(context.Background(), input)
	require.EqualError(t, err, "failed to exchange code: temporary provider failure")
	_, ok := svc.sessionStore.Get(sessionID)
	require.True(t, ok, "a failed Gemini exchange must leave the session retryable")

	tokenInfo, err := svc.ExchangeCode(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "gemini-retry-access-token", tokenInfo.AccessToken)
	require.Equal(t, 2, calls)
	_, ok = svc.sessionStore.Get(sessionID)
	require.False(t, ok, "a successful Gemini retry must consume the session")
}
