//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"anlapi/internal/pkg/oauth"
	"github.com/stretchr/testify/require"
)

func TestOAuthService_ExchangeCode_RetryAfterProviderFailureKeepsSession(t *testing.T) {
	t.Parallel()

	calls := 0
	client := &mockClaudeOAuthClient{
		exchangeCodeFunc: func(context.Context, string, string, string, string, bool) (*oauth.TokenResponse, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("temporary provider failure")
			}
			return &oauth.TokenResponse{
				AccessToken:  "retry-access-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "retry-refresh-token",
				Scope:        oauth.ScopeOAuth,
			}, nil
		},
	}

	svc := NewOAuthService(&mockProxyRepoForOAuth{}, client)
	defer svc.Stop()

	authURL, err := svc.GenerateAuthURL(context.Background(), nil)
	require.NoError(t, err)

	_, err = svc.ExchangeCode(context.Background(), &ExchangeCodeInput{
		SessionID: authURL.SessionID,
		Code:      "retry-code",
	})
	require.EqualError(t, err, "temporary provider failure")
	_, ok := svc.sessionStore.Get(authURL.SessionID)
	require.True(t, ok, "a failed provider exchange must leave the session retryable")

	tokenInfo, err := svc.ExchangeCode(context.Background(), &ExchangeCodeInput{
		SessionID: authURL.SessionID,
		Code:      "retry-code",
	})
	require.NoError(t, err)
	require.Equal(t, "retry-access-token", tokenInfo.AccessToken)
	require.Equal(t, 2, calls)
	_, ok = svc.sessionStore.Get(authURL.SessionID)
	require.False(t, ok, "a successful retry must consume the session")
}
