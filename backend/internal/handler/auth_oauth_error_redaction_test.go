package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateLogValueRedactsOAuthSecrets(t *testing.T) {
	got := truncateLogValue(`{"error":"invalid_grant","access_token":"access-secret","refresh_token":"refresh-secret","client_secret":"client-secret"}`, 1024)

	require.Contains(t, got, `"access_token":"***"`)
	require.NotContains(t, got, "access-secret")
	require.NotContains(t, got, "refresh-secret")
	require.NotContains(t, got, "client-secret")
}

func TestTruncateLogValueStillHonorsLimitAfterRedaction(t *testing.T) {
	got := truncateLogValue(`{"message":"`+string(make([]byte, 2048))+`"}`, 64)
	require.LessOrEqual(t, len(got), 64)
}
