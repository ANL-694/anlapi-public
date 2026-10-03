package service

import (
	"testing"

	"anlapi/internal/pkg/typesafe"
	"github.com/stretchr/testify/require"
)

func TestTypeSafeAccountCredentialsUseNativeDefaults(t *testing.T) {
	account := &Account{Platform: PlatformTypeSafe, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": " ts-secret "}}
	require.True(t, account.IsTypeSafe())
	require.Equal(t, typesafe.DefaultBaseURL, account.GetBaseURL())
	require.Equal(t, typesafe.DefaultBaseURL, account.GetTypeSafeBaseURL())
	require.Equal(t, "ts-secret", account.GetTypeSafeAPIKey())

	for raw, want := range map[string]string{
		"https://api.typesafe.ai/v1":      "https://api.typesafe.ai",
		"https://api.typesafe.ai/V1/":     "https://api.typesafe.ai",
		"https://proxy.example/typesafe/": "https://proxy.example/typesafe",
		"https://proxy.example/apiv1":     "https://proxy.example/apiv1",
		" https://proxy.example/x/v1/ ":   "https://proxy.example/x",
		"/v1":                             typesafe.DefaultBaseURL,
	} {
		withBase := &Account{Platform: PlatformTypeSafe, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": raw}}
		require.Equal(t, want, withBase.GetTypeSafeBaseURL(), raw)
	}
}

func TestTypeSafeAccountDoesNotLeakIntoOtherCredentialTypes(t *testing.T) {
	for _, account := range []*Account{
		nil,
		{Platform: PlatformTypeSafe, Type: AccountTypeOAuth, Credentials: map[string]any{"api_key": "secret"}},
		{Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "secret"}},
	} {
		require.Empty(t, account.GetTypeSafeBaseURL())
		require.Empty(t, account.GetTypeSafeAPIKey())
	}
	require.Equal(t, "https://api.anthropic.com", (&Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}).GetBaseURL())
}
