package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"anlapi/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type affiliateWithdrawHandlerRepoStub struct {
	service.AffiliateRepository
	result      *service.AffiliateWithdrawResult
	called      bool
	userID      int64
	amount      float64
	operationID string
	overview    *service.AffiliateUserOverview
}

func (s *affiliateWithdrawHandlerRepoStub) WithdrawQuota(_ context.Context, userID int64, amount float64, operationID string) (*service.AffiliateWithdrawResult, error) {
	s.called = true
	s.userID = userID
	s.amount = amount
	s.operationID = operationID
	return s.result, nil
}

func (s *affiliateWithdrawHandlerRepoStub) GetAffiliateUserOverview(context.Context, int64) (*service.AffiliateUserOverview, error) {
	return s.overview, nil
}

type affiliateWithdrawResponse struct {
	Code   int             `json:"code"`
	Reason string          `json:"reason"`
	Data   json.RawMessage `json:"data"`
}

func performAffiliateWithdrawRequest(t *testing.T, repo *affiliateWithdrawHandlerRepoStub, userID, key, body string) (*httptest.ResponseRecorder, affiliateWithdrawResponse) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), nil)
	router.POST("/api/v1/admin/affiliates/users/:user_id/withdraw", h.WithdrawQuota)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/affiliates/users/42/withdraw", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var response affiliateWithdrawResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	return rec, response
}

func TestAffiliateHandlerWithdrawQuotaSuccessAndReplayHeader(t *testing.T) {
	repo := &affiliateWithdrawHandlerRepoStub{result: &service.AffiliateWithdrawResult{
		LedgerID:            9,
		UserID:              42,
		Amount:              10,
		AvailableQuotaAfter: 90,
		Replayed:            true,
	}}

	rec, response := performAffiliateWithdrawRequest(t, repo, "42", "offline-withdraw-1", `{"amount":10}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 0, response.Code)
	require.Equal(t, "true", rec.Header().Get("X-Idempotency-Replayed"))
	require.True(t, repo.called)
	require.Equal(t, int64(42), repo.userID)
	require.Equal(t, 10.0, repo.amount)
	require.NotEmpty(t, repo.operationID)
}

func TestAffiliateHandlerWithdrawQuotaRejectsMissingKeyBeforeRepository(t *testing.T) {
	repo := &affiliateWithdrawHandlerRepoStub{}

	rec, response := performAffiliateWithdrawRequest(t, repo, "42", "", `{"amount":10}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "IDEMPOTENCY_KEY_REQUIRED", response.Reason)
	require.False(t, repo.called)
}

func TestAffiliateHandlerGetUserOverviewReturnsReadOnlySummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &affiliateWithdrawHandlerRepoStub{overview: &service.AffiliateUserOverview{
		UserID:         42,
		AvailableQuota: 12.5,
		HistoryQuota:   20,
	}}
	router := gin.New()
	h := NewAffiliateHandler(service.NewAffiliateService(repo, nil, nil, nil), nil)
	router.GET("/api/v1/admin/affiliates/users/:user_id/overview", h.GetUserOverview)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/affiliates/users/42/overview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var response affiliateWithdrawResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 0, response.Code)
	require.Contains(t, string(response.Data), `"available_quota":12.5`)
}
