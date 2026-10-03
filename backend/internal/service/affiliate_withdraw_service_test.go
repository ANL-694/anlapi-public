//go:build unit

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"

	infraerrors "anlapi/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type affiliateWithdrawRepoStub struct {
	AffiliateRepository
	operationID string
	userID      int64
	amount      float64
	result      *AffiliateWithdrawResult
	err         error
	overview    *AffiliateUserOverview
}

func (s *affiliateWithdrawRepoStub) WithdrawQuota(_ context.Context, userID int64, amount float64, operationID string) (*AffiliateWithdrawResult, error) {
	s.userID = userID
	s.amount = amount
	s.operationID = operationID
	return s.result, s.err
}

func (s *affiliateWithdrawRepoStub) GetAffiliateUserOverview(context.Context, int64) (*AffiliateUserOverview, error) {
	return s.overview, nil
}

func TestAdminWithdrawQuota_NormalizesAmountAndHashesKey(t *testing.T) {
	repo := &affiliateWithdrawRepoStub{result: &AffiliateWithdrawResult{LedgerID: 9, UserID: 42, Amount: 1.23456789}}
	svc := NewAffiliateService(repo, nil, nil, nil)

	result, err := svc.AdminWithdrawQuota(context.Background(), 42, 1.234567891, "  offline-42  ")

	require.NoError(t, err)
	require.Equal(t, repo.result, result)
	require.Equal(t, int64(42), repo.userID)
	require.Equal(t, 1.23456789, repo.amount)
	sum := sha256.Sum256([]byte("admin.affiliates.withdraw\x00offline-42"))
	require.Equal(t, hex.EncodeToString(sum[:]), repo.operationID)
}

func TestAdminWithdrawQuotaRejectsInvalidRequestsBeforeRepository(t *testing.T) {
	cases := []struct {
		name   string
		userID int64
		amount float64
		key    string
		code   string
	}{
		{name: "invalid user", userID: 0, amount: 1, key: "valid-key", code: "INVALID_USER"},
		{name: "missing key", userID: 1, amount: 1, code: "IDEMPOTENCY_KEY_REQUIRED"},
		{name: "invalid key", userID: 1, amount: 1, key: "bad\nkey", code: "IDEMPOTENCY_KEY_INVALID"},
		{name: "zero amount", userID: 1, amount: 0, key: "valid-key", code: "AFFILIATE_WITHDRAW_AMOUNT_INVALID"},
		{name: "nan amount", userID: 1, amount: math.NaN(), key: "valid-key", code: "AFFILIATE_WITHDRAW_AMOUNT_INVALID"},
		{name: "infinite amount", userID: 1, amount: math.Inf(1), key: "valid-key", code: "AFFILIATE_WITHDRAW_AMOUNT_INVALID"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &affiliateWithdrawRepoStub{}
			svc := NewAffiliateService(repo, nil, nil, nil)

			_, err := svc.AdminWithdrawQuota(context.Background(), tc.userID, tc.amount, tc.key)

			require.Error(t, err)
			require.Equal(t, tc.code, infraerrors.Reason(err))
			require.Zero(t, repo.userID)
			require.Empty(t, repo.operationID)
		})
	}
}

func TestAdminWithdrawQuotaPropagatesRepositoryError(t *testing.T) {
	repoErr := ErrAffiliateQuotaInsufficient
	repo := &affiliateWithdrawRepoStub{err: repoErr}
	svc := NewAffiliateService(repo, nil, nil, nil)

	_, err := svc.AdminWithdrawQuota(context.Background(), 42, 1, "valid-key")

	require.ErrorIs(t, err, repoErr)
}

func TestAdminGetUserOverviewUsesGlobalRateWhenUserRateIsNotCustom(t *testing.T) {
	repo := &affiliateWithdrawRepoStub{overview: &AffiliateUserOverview{UserID: 42, AvailableQuota: 3.5}}
	svc := NewAffiliateService(repo, nil, nil, nil)

	overview, err := svc.AdminGetUserOverview(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 3.5, overview.AvailableQuota)
	require.Equal(t, AffiliateRebateRateDefault, overview.RebateRatePercent)
}
