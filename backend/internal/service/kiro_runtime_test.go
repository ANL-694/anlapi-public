package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStableKiroConversationIDUsesAnlapiEnvironmentVariable(t *testing.T) {
	account := &Account{ID: 1}
	parsed := &ParsedRequest{ExplicitSessionID: "session-1"}
	t.Setenv("ANLAPI_KIRO_CONVERSATION_ID_MODE", "")
	t.Setenv("LEGACY_KIRO_CONVERSATION_ID_MODE", "random")

	legacyIgnored := stableKiroConversationID(account, parsed, nil, "model", "")
	require.NotEmpty(t, legacyIgnored)
	require.Equal(t, legacyIgnored, stableKiroConversationID(account, parsed, nil, "model", ""))

	t.Setenv("ANLAPI_KIRO_CONVERSATION_ID_MODE", "random")
	require.Empty(t, stableKiroConversationID(account, parsed, nil, "model", ""))
}
