package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"anlapi/ent/authidentity"
	dbuser "anlapi/ent/user"
	"anlapi/internal/config"
	"anlapi/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExchangePendingOAuthChoiceStateDoesNotBindIdentityAcrossProviders(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, false)
	ctx := context.Background()

	victim, err := client.User.Create().
		SetEmail("choice-victim@example.com").
		SetUsername("choice-victim").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	providers := []struct {
		name string
		key  string
	}{
		{name: "linuxdo", key: "linuxdo"},
		{name: "oidc", key: "https://issuer.example.com"},
		{name: "wechat", key: "wechat"},
		{name: "dingtalk", key: "dingtalk"},
		{name: "github", key: "github"},
		{name: "google", key: "google"},
	}

	for _, provider := range providers {
		t.Run(provider.name, func(t *testing.T) {
			subject := provider.name + "-attacker-subject"
			session, err := client.PendingAuthSession.Create().
				SetSessionToken(subject + "-session-token").
				SetIntent("login").
				SetProviderType(provider.name).
				SetProviderKey(provider.key).
				SetProviderSubject(subject).
				SetTargetUserID(victim.ID).
				SetResolvedEmail(victim.Email).
				SetBrowserSessionKey(subject + "-browser-key").
				SetLocalFlowState(map[string]any{
					oauthCompletionResponseKey: map[string]any{
						"step":                   oauthPendingChoiceStep,
						"adoption_required":      true,
						"email_binding_required": true,
						"email":                  victim.Email,
					},
				}).
				SetExpiresAt(time.Now().UTC().Add(10 * time.Minute)).
				Save(ctx)
			require.NoError(t, err)

			body := bytes.NewBufferString(`{"adopt_display_name":true,"adopt_avatar":true}`)
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/pending/exchange", body)
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: oauthPendingSessionCookieName, Value: encodeCookieValue(session.SessionToken)})
			req.AddCookie(&http.Cookie{Name: oauthPendingBrowserCookieName, Value: encodeCookieValue(subject + "-browser-key")})
			ginCtx.Request = req

			handler.ExchangePendingOAuthCompletion(ginCtx)

			require.Equal(t, http.StatusOK, recorder.Code)
			payload := decodeJSONResponseData(t, recorder)
			require.Equal(t, oauthPendingChoiceStep, payload["step"])
			require.NotContains(t, payload, "access_token")

			identityCount, err := client.AuthIdentity.Query().Where(
				authidentity.ProviderTypeEQ(provider.name),
				authidentity.ProviderKeyEQ(provider.key),
				authidentity.ProviderSubjectEQ(subject),
			).Count(ctx)
			require.NoError(t, err)
			require.Zero(t, identityCount)

			stored, err := client.PendingAuthSession.Get(ctx, session.ID)
			require.NoError(t, err)
			require.Nil(t, stored.ConsumedAt)
		})
	}
}

func TestCompleteDingTalkOAuthRegistrationRejectsIdentityOwnershipConflict(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, false)
	ctx := context.Background()

	owner, err := client.User.Create().
		SetEmail("dingtalk-owner@example.com").
		SetUsername("dingtalk-owner").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.AuthIdentity.Create().
		SetUserID(owner.ID).
		SetProviderType("dingtalk").
		SetProviderKey("dingtalk").
		SetProviderSubject("dingtalk-conflict-subject").
		Save(ctx)
	require.NoError(t, err)

	session, err := client.PendingAuthSession.Create().
		SetSessionToken("dingtalk-conflict-session-token").
		SetIntent("login").
		SetProviderType("dingtalk").
		SetProviderKey("dingtalk").
		SetProviderSubject("dingtalk-conflict-subject").
		SetResolvedEmail("dingtalk-new@example.com").
		SetBrowserSessionKey("dingtalk-conflict-browser-key").
		SetUpstreamIdentityClaims(map[string]any{"username": "dingtalk-new-user"}).
		SetExpiresAt(time.Now().UTC().Add(10 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/dingtalk/complete-registration", bytes.NewBufferString(`{"invitation_code":"invite-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: oauthPendingSessionCookieName, Value: encodeCookieValue(session.SessionToken)})
	req.AddCookie(&http.Cookie{Name: oauthPendingBrowserCookieName, Value: encodeCookieValue("dingtalk-conflict-browser-key")})
	ginCtx.Request = req

	handler.CompleteDingTalkOAuthRegistration(ginCtx)

	require.Equal(t, http.StatusConflict, recorder.Code)
	payload := decodeJSONBody(t, recorder)
	require.Equal(t, "AUTH_IDENTITY_OWNERSHIP_CONFLICT", payload["reason"])

	userCount, err := client.User.Query().Where(dbuser.EmailEQ("dingtalk-new@example.com")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, userCount)
	stored, err := client.PendingAuthSession.Get(ctx, session.ID)
	require.NoError(t, err)
	require.Nil(t, stored.ConsumedAt)
}

func TestEmailOAuthCallbackRejectsIdentityLinkedToDifferentEmail(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, false)
	ctx := context.Background()

	for _, provider := range []string{"github", "google"} {
		t.Run(provider, func(t *testing.T) {
			subject := provider + "-owner-subject"
			owner, err := client.User.Create().
				SetEmail(provider + "-owner@example.com").
				SetUsername(provider + "-owner").
				SetPasswordHash("hash").
				SetRole(service.RoleUser).
				SetStatus(service.StatusActive).
				Save(ctx)
			require.NoError(t, err)

			_, err = client.AuthIdentity.Create().
				SetUserID(owner.ID).
				SetProviderType(provider).
				SetProviderKey(provider).
				SetProviderSubject(subject).
				Save(ctx)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/"+provider+"/callback", nil)
			handler.emailOAuthCallbackWithProfile(ginCtx, provider, config.EmailOAuthProviderConfig{
				FrontendRedirectURL: "/auth/oauth/callback",
			}, "/auth/oauth/callback", "/dashboard", &emailOAuthProfile{
				Subject:       subject,
				Email:         "different-email@example.com",
				EmailVerified: true,
				Username:      "different-user",
			})

			require.Equal(t, http.StatusFound, recorder.Code)
			assertOAuthRedirectError(t, recorder.Header().Get("Location"), "AUTH_IDENTITY_EMAIL_MISMATCH", "oauth identity belongs to a different email")
			userCount, err := client.User.Query().Where(dbuser.EmailEQ("different-email@example.com")).Count(ctx)
			require.NoError(t, err)
			require.Zero(t, userCount)
			identity, err := client.AuthIdentity.Query().Where(
				authidentity.ProviderTypeEQ(provider),
				authidentity.ProviderKeyEQ(provider),
				authidentity.ProviderSubjectEQ(subject),
			).Only(ctx)
			require.NoError(t, err)
			require.Equal(t, owner.ID, identity.UserID)
			count, err := client.PendingAuthSession.Query().Count(ctx)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}
