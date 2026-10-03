package service

import (
	"context"
	"testing"
	"time"

	"anlapi/internal/pkg/antigravity"
	kiropkg "anlapi/internal/pkg/kiro"
	"github.com/stretchr/testify/require"
)

func TestKiroOAuthService_ExchangeCodeGuardsStateAndSession(t *testing.T) {
	t.Parallel()

	svc := NewKiroOAuthService(nil)
	input := &KiroExchangeCodeInput{SessionID: "missing", State: "state", Code: "code"}
	_, err := svc.ExchangeCode(context.Background(), input)
	require.EqualError(t, err, "session not found or expired")

	const sessionID = "kiro-guard-session"
	svc.sessionStore.Set(sessionID, &kiropkg.AuthSession{
		State:        "expected-state",
		CodeVerifier: "verifier",
		CreatedAt:    time.Now(),
		AuthType:     "social",
	})

	input.SessionID = sessionID
	input.State = "wrong-state"
	_, err = svc.ExchangeCode(context.Background(), input)
	require.EqualError(t, err, "state invalid")
	_, ok := svc.sessionStore.Get(sessionID)
	require.True(t, ok, "invalid state must not consume the pending session")
}

func TestAntigravityOAuthService_ExchangeCodeGuardsStateAndSession(t *testing.T) {
	t.Parallel()

	svc := NewAntigravityOAuthService(nil)
	defer svc.Stop()
	input := &AntigravityExchangeCodeInput{SessionID: "missing", State: "state", Code: "code"}
	_, err := svc.ExchangeCode(context.Background(), input)
	require.EqualError(t, err, "session 不存在或已过期")

	const sessionID = "antigravity-guard-session"
	svc.sessionStore.Set(sessionID, &antigravity.OAuthSession{
		State:        "expected-state",
		CodeVerifier: "verifier",
		CreatedAt:    time.Now(),
	})

	input.SessionID = sessionID
	input.State = "wrong-state"
	_, err = svc.ExchangeCode(context.Background(), input)
	require.EqualError(t, err, "state 无效")
	_, ok := svc.sessionStore.Get(sessionID)
	require.True(t, ok, "invalid state must not consume the pending session")
}
