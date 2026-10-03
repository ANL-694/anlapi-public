package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeStoredCredentialsRemovesTemporaryLoginSecrets(t *testing.T) {
	credentials := map[string]any{
		"access_token":      "fresh-access-token",
		"refresh_token":     "fresh-refresh-token",
		"model_mapping":     map[string]any{"gpt-5": "gpt-5"},
		"password":          "temporary-password",
		"sso_token":         "temporary-sso-token",
		"sso":               "temporary-sso",
		"sso-rw":            "temporary-sso-rw",
		"clearTextPassword": "temporary-plaintext-password",
		"cookie":            "temporary-cookie",
	}

	got := SanitizeStoredCredentials(PlatformOpenAI, credentials)

	require.Equal(t, "fresh-access-token", got["access_token"])
	require.Equal(t, "fresh-refresh-token", got["refresh_token"])
	require.Equal(t, map[string]any{"gpt-5": "gpt-5"}, got["model_mapping"])
	for _, key := range []string{"password", "sso_token", "sso", "sso-rw", "clearTextPassword", "cookie"} {
		require.NotContains(t, got, key)
	}
}

func TestSanitizeStoredCredentialsAllowsEmptyPlatformAndNil(t *testing.T) {
	got := SanitizeStoredCredentials("", map[string]any{"cookie": "temporary-cookie", "api_key": "retained"})
	require.NotContains(t, got, "cookie")
	require.Equal(t, "retained", got["api_key"])
	require.Nil(t, SanitizeStoredCredentials(PlatformOpenAI, nil))
}
