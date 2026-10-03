//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"anlapi/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAffiliateRepository_WithdrawQuotaIsIdempotent(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewAffiliateRepository(client, integrationDB)

	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-withdraw-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      3.5,
		Concurrency:  1,
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", u.ID)
	})
	affCode := fmt.Sprintf("WD%010d", time.Now().UnixNano()%10_000_000_000)
	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, $3, $3, NOW(), NOW())`, u.ID, affCode, 12.34567891)
	require.NoError(t, err)

	first, err := repo.WithdrawQuota(ctx, u.ID, 2.5, "withdraw-operation-1")
	require.NoError(t, err)
	require.False(t, first.Replayed)
	require.Equal(t, 2.5, first.Amount)
	require.InDelta(t, 9.84567891, first.AvailableQuotaAfter, 1e-8)

	replay, err := repo.WithdrawQuota(ctx, u.ID, 2.5, "withdraw-operation-1")
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, first.LedgerID, replay.LedgerID)
	require.Equal(t, first.AvailableQuotaAfter, replay.AvailableQuotaAfter)

	_, err = repo.WithdrawQuota(ctx, u.ID, 3.5, "withdraw-operation-1")
	require.ErrorIs(t, err, service.ErrIdempotencyKeyConflict)

	_, err = repo.WithdrawQuota(ctx, u.ID, 20, "withdraw-operation-insufficient")
	require.ErrorIs(t, err, service.ErrAffiliateQuotaInsufficient)
	var placeholderCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM user_affiliate_ledger WHERE operation_id = $1", "withdraw-operation-insufficient").Scan(&placeholderCount))
	require.Zero(t, placeholderCount)

	var quota float64
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID).Scan(&quota))
	require.InDelta(t, 9.84567891, quota, 1e-8)
}

func TestAffiliateRepository_WithdrawQuotaConcurrentSameKeyDeductsOnce(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewAffiliateRepository(client, integrationDB)
	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-withdraw-concurrent-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      1,
		Concurrency:  1,
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", u.ID)
	})
	affCode := fmt.Sprintf("WC%010d", time.Now().UnixNano()%10_000_000_000)
	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, $3, $3, NOW(), NOW())`, u.ID, affCode, 7.5)
	require.NoError(t, err)

	results := make([]*service.AffiliateWithdrawResult, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errs[index] = repo.WithdrawQuota(ctx, u.ID, 2.25, "withdraw-operation-concurrent")
		}(i)
	}
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.NotNil(t, results[0])
	require.NotNil(t, results[1])
	require.NotEqual(t, results[0].Replayed, results[1].Replayed)
	require.Equal(t, results[0].LedgerID, results[1].LedgerID)

	var quota float64
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", u.ID).Scan(&quota))
	require.InDelta(t, 5.25, quota, 1e-8)
}

func TestAffiliateRepository_GetAffiliateUserOverviewDoesNotCreateProfile(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewAffiliateRepository(client, integrationDB)
	u := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-overview-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      1,
		Concurrency:  1,
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", u.ID)
	})
	affCode := fmt.Sprintf("OV%010d", time.Now().UnixNano()%10_000_000_000)
	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, $3, $3, NOW(), NOW())`, u.ID, affCode, 4.25)
	require.NoError(t, err)

	overview, err := repo.GetAffiliateUserOverview(ctx, u.ID)

	require.NoError(t, err)
	require.Equal(t, u.ID, overview.UserID)
	require.Equal(t, affCode, overview.AffCode)
	require.InDelta(t, 4.25, overview.AvailableQuota, 1e-8)

	missingUserID := u.ID + 9_000_000_000
	_, err = repo.GetAffiliateUserOverview(ctx, missingUserID)
	require.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestAffiliateRepository_ListAffiliateRecordsPreservesHistoricalNulls(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewAffiliateRepository(client, integrationDB)

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-record-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      1,
		Concurrency:  1,
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-record-invitee-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      1,
		Concurrency:  1,
	})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id IN ($1, $2)", inviter.ID, invitee.ID)
	})

	inviterCode := fmt.Sprintf("RI%010d", time.Now().UnixNano()%10_000_000_000)
	inviteeCode := fmt.Sprintf("RE%010d", time.Now().UnixNano()%10_000_000_000)
	_, err := integrationDB.ExecContext(ctx, `
INSERT INTO user_affiliates (user_id, aff_code, inviter_id, aff_count, created_at, updated_at)
VALUES ($1, $2, NULL, 1, NOW(), NOW()), ($3, $4, $1, 0, NOW(), NOW())`,
		inviter.ID, inviterCode, invitee.ID, inviteeCode)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, created_at, updated_at)
VALUES ($1, 'accrue', 1.25, $2, NOW(), NOW()), ($1, 'transfer', 0.5, NULL, NOW(), NOW())`, inviter.ID, invitee.ID)
	require.NoError(t, err)

	filter := service.AffiliateRecordFilter{Page: 1, PageSize: 20, Search: inviter.Email}
	invites, inviteTotal, err := repo.ListAffiliateInviteRecords(ctx, filter)
	require.NoError(t, err)
	require.EqualValues(t, 1, inviteTotal)
	require.Len(t, invites, 1)
	require.Equal(t, invitee.ID, invites[0].InviteeID)

	rebates, rebateTotal, err := repo.ListAffiliateRebateRecords(ctx, service.AffiliateRecordFilter{Page: 1, PageSize: 20, Search: inviter.Email})
	require.NoError(t, err)
	require.EqualValues(t, 1, rebateTotal)
	require.Len(t, rebates, 1)
	require.Nil(t, rebates[0].OrderID)
	require.Nil(t, rebates[0].OrderAmount)
	require.Equal(t, invitee.ID, *rebates[0].InviteeID)

	transfers, transferTotal, err := repo.ListAffiliateTransferRecords(ctx, service.AffiliateRecordFilter{Page: 1, PageSize: 20, Search: inviter.Email})
	require.NoError(t, err)
	require.EqualValues(t, 1, transferTotal)
	require.Len(t, transfers, 1)
	require.Equal(t, "transfer", transfers[0].Action)
	require.False(t, transfers[0].SnapshotAvailable)
}
