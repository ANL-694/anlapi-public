package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateAccountSanitizesCredentialsAfterOAuthReauthMerge(t *testing.T) {
	const accountID = int64(501)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
		accountID: {
			ID:       accountID,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token":  "old-access-token",
				"refresh_token": "old-refresh-token",
				"model_mapping": map[string]any{"gpt-5": "gpt-5"},
				"password":      "temporary-password",
				"sso_token":     "temporary-sso-token",
				"cookie":        "temporary-cookie",
			},
		},
	}}

	updated, err := (&adminServiceImpl{accountRepo: repo}).UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
		Type: AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "fresh-access-token",
			"refresh_token": "fresh-refresh-token",
			"model_mapping": map[string]any{"gpt-5": "gpt-5"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "fresh-access-token", updated.Credentials["access_token"])
	require.Equal(t, "fresh-refresh-token", updated.Credentials["refresh_token"])
	require.Equal(t, map[string]any{"gpt-5": "gpt-5"}, updated.Credentials["model_mapping"])
	for _, key := range []string{"password", "sso_token", "cookie"} {
		require.NotContains(t, updated.Credentials, key)
	}
}

func TestUpdateAccountDoesNotSanitizeCredentialsDuringOrdinaryEdit(t *testing.T) {
	const accountID = int64(502)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
		accountID: {
			ID:       accountID,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token": "existing-access-token",
				"cookie":       "existing-cookie",
			},
		},
	}}

	updated, err := (&adminServiceImpl{accountRepo: repo}).UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5": "gpt-5"}},
	})

	require.NoError(t, err)
	require.Equal(t, "existing-access-token", updated.Credentials["access_token"])
	require.Equal(t, "existing-cookie", updated.Credentials["cookie"])
}
