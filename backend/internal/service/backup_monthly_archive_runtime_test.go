//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	infraerrors "anlapi/internal/pkg/errors"
)

func monthlyArchiveTestSchedule() *BackupScheduleConfig {
	return &BackupScheduleConfig{
		CronExpr:    "0 4 * * *",
		RetainCount: 1,
		MonthlyArchive: &BackupMonthlyArchiveConfig{
			Enabled:         true,
			Days:            []int{1, 15},
			IncludeMonthEnd: true,
			RetainCount:     1,
		},
	}
}

func TestBackupMonthlyArchiveRuntime_AssignsAndAdvancesCheckpoint(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	schedule := monthlyArchiveTestSchedule()

	record := &BackupRecord{
		ID:          "scheduled-1",
		Status:      "completed",
		TriggeredBy: "scheduled",
		StartedAt:   "2026-09-15T04:00:00Z",
		ExpiresAt:   "2026-09-29T04:00:00Z",
	}
	require.NoError(t, svc.saveRecordWithArchive(context.Background(), record, schedule))
	require.Equal(t, []string{"2026-09-01", "2026-09-15"}, record.MonthlyArchive.Dates)
	require.Empty(t, record.ExpiresAt)
	require.Equal(t, 1, record.MonthlyArchive.RetainCount)

	checkpoint, err := repo.GetValue(context.Background(), settingKeyBackupArchiveCheckpoint)
	require.NoError(t, err)
	require.Equal(t, "2026-09-15", checkpoint)

	// A retry of the completed record is idempotent and does not change metadata.
	require.NoError(t, svc.saveRecordWithArchive(context.Background(), record, schedule))
	retryCheckpoint, err := repo.GetValue(context.Background(), settingKeyBackupArchiveCheckpoint)
	require.NoError(t, err)
	require.Equal(t, checkpoint, retryCheckpoint)
}

func TestBackupMonthlyArchiveRuntime_AdvancesWithoutDueDate(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupArchiveCheckpoint, "2026-09-15"))

	record := &BackupRecord{
		ID:          "scheduled-2",
		Status:      "completed",
		TriggeredBy: "scheduled",
		StartedAt:   "2026-09-20T04:00:00Z",
	}
	require.NoError(t, svc.saveRecordWithArchive(context.Background(), record, monthlyArchiveTestSchedule()))
	require.Nil(t, record.MonthlyArchive)
	checkpoint, err := repo.GetValue(context.Background(), settingKeyBackupArchiveCheckpoint)
	require.NoError(t, err)
	require.Equal(t, "2026-09-20", checkpoint)
}

func TestBackupMonthlyArchiveRuntime_RejectsCorruptCheckpoint(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupArchiveCheckpoint, "not-a-date"))

	record := &BackupRecord{
		ID:          "scheduled-corrupt",
		Status:      "completed",
		TriggeredBy: "scheduled",
		StartedAt:   "2026-09-20T04:00:00Z",
	}
	err := svc.saveRecordWithArchive(context.Background(), record, monthlyArchiveTestSchedule())
	require.Equal(t, "BACKUP_ARCHIVE_CHECKPOINT_CORRUPT", infraerrors.Reason(err))
}

func TestBackupMonthlyArchiveRuntime_CleanupUsesSeparatePools(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())

	records := []BackupRecord{
		{ID: "ordinary-new", Status: "completed", StartedAt: "2026-09-20T04:00:00Z", S3Key: "ordinary-new"},
		{ID: "ordinary-old", Status: "completed", StartedAt: "2026-09-10T04:00:00Z", S3Key: "ordinary-old"},
		{ID: "archive-new", Status: "completed", StartedAt: "2026-09-19T04:00:00Z", S3Key: "archive-new", MonthlyArchive: &BackupMonthlyArchive{Dates: []string{"2026-09-15"}, RetainCount: 1}},
		{ID: "archive-old", Status: "completed", StartedAt: "2026-09-09T04:00:00Z", S3Key: "archive-old", MonthlyArchive: &BackupMonthlyArchive{Dates: []string{"2026-09-01"}, RetainCount: 1}},
		{ID: "archive-permanent", Status: "completed", StartedAt: "2026-09-01T04:00:00Z", S3Key: "archive-permanent", MonthlyArchive: &BackupMonthlyArchive{Dates: []string{"2026-08-31"}, RetainCount: 0}},
	}
	for i := range records {
		require.NoError(t, svc.saveRecord(context.Background(), &records[i]))
	}

	require.NoError(t, svc.cleanupOldBackups(context.Background(), monthlyArchiveTestSchedule()))
	remaining, err := svc.ListBackups(context.Background())
	require.NoError(t, err)
	ids := make(map[string]bool, len(remaining))
	for _, record := range remaining {
		ids[record.ID] = true
	}
	require.True(t, ids["ordinary-new"])
	require.False(t, ids["ordinary-old"])
	require.True(t, ids["archive-new"])
	require.False(t, ids["archive-old"])
	require.True(t, ids["archive-permanent"])
}

func TestBackupMonthlyArchiveRuntime_RequiresExplicitDelete(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	record := &BackupRecord{
		ID:             "archive-delete",
		Status:         "completed",
		S3Key:          "archive-delete",
		MonthlyArchive: &BackupMonthlyArchive{Dates: []string{"2026-09-01"}, RetainCount: 0},
	}
	require.NoError(t, svc.saveRecord(context.Background(), record))
	require.Equal(t, "BACKUP_ARCHIVE_PROTECTED", infraerrors.Reason(svc.DeleteBackup(context.Background(), record.ID)))
	require.NoError(t, svc.DeleteArchivedBackup(context.Background(), record.ID))
	_, err := svc.GetBackupRecord(context.Background(), record.ID)
	require.Equal(t, "BACKUP_NOT_FOUND", infraerrors.Reason(err))

	// Stored JSON remains valid after the explicit deletion path.
	var stored []BackupRecord
	raw, err := repo.GetValue(context.Background(), settingKeyBackupRecords)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	require.Empty(t, stored)
}

func TestBackupMonthlyArchiveRuntime_LeaderLockRejectsPeerWithoutMutation(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	cache := &fakeLeaderLockCache{}
	svc.SetLeaderLock(cache, nil)

	initial := &BackupRecord{
		ID:             "archive-existing",
		Status:         "completed",
		StartedAt:      "2026-09-01T04:00:00Z",
		MonthlyArchive: &BackupMonthlyArchive{Dates: []string{"2026-09-01"}, RetainCount: 1},
	}
	require.NoError(t, svc.saveRecord(context.Background(), initial))
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupArchiveCheckpoint, "2026-09-01"))
	rawBefore, err := repo.GetValue(context.Background(), settingKeyBackupRecords)
	require.NoError(t, err)
	checkpointBefore, err := repo.GetValue(context.Background(), settingKeyBackupArchiveCheckpoint)
	require.NoError(t, err)

	releasePeer, acquired, err := acquireTestBackupRecordsLock(cache, "peer")
	require.NoError(t, err)
	require.True(t, acquired)
	defer releasePeer()

	newRecord := &BackupRecord{
		ID:          "archive-new",
		Status:      "completed",
		TriggeredBy: "scheduled",
		StartedAt:   "2026-09-15T04:00:00Z",
	}
	err = svc.saveRecordWithArchive(context.Background(), newRecord, monthlyArchiveTestSchedule())
	require.ErrorIs(t, err, ErrBackupRecordsBusy)

	rawAfter, err := repo.GetValue(context.Background(), settingKeyBackupRecords)
	require.NoError(t, err)
	checkpointAfter, err := repo.GetValue(context.Background(), settingKeyBackupArchiveCheckpoint)
	require.NoError(t, err)
	require.Equal(t, rawBefore, rawAfter)
	require.Equal(t, checkpointBefore, checkpointAfter)
}

func TestBackupMonthlyArchiveRuntime_LeaderLockReleasesAndFailsClosed(t *testing.T) {
	repo := newMockSettingRepo()
	cache := &fakeLeaderLockCache{}
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	svc.SetLeaderLock(cache, nil)

	record := &BackupRecord{ID: "lock-release", Status: "completed", StartedAt: time.Now().UTC().Format(time.RFC3339)}
	require.NoError(t, svc.saveRecord(context.Background(), record))
	require.Empty(t, cache.heldBy(backupRecordsLeaderLockKey))

	cache.acquireErr = context.DeadlineExceeded
	err := svc.saveRecord(context.Background(), &BackupRecord{ID: "lock-error", Status: "completed"})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	stored, err := svc.ListBackups(context.Background())
	require.NoError(t, err)
	require.Len(t, stored, 1)
	require.Equal(t, "lock-release", stored[0].ID)
}

func acquireTestBackupRecordsLock(cache *fakeLeaderLockCache, owner string) (func(), bool, error) {
	return (&BackupService{lockCache: cache, instanceID: owner}).tryAcquireRecordsLock(context.Background())
}

type blockingBackupLeaderLockCache struct{}

func (blockingBackupLeaderLockCache) TryAcquireLeaderLock(ctx context.Context, _ string, _ string, _ time.Duration) (bool, error) {
	<-ctx.Done()
	return false, ctx.Err()
}

func (blockingBackupLeaderLockCache) ReleaseLeaderLock(context.Context, string, string) error {
	return nil
}

func TestBackupMonthlyArchiveRuntime_LeaderLockUsesPostgresFallback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	lockID := hashAdvisoryLockID(backupRecordsLeaderLockKey)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).WithArgs(lockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectExec(`SELECT pg_advisory_unlock\(\$1\)`).WithArgs(lockID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := newTestBackupService(newMockSettingRepo(), &mockDumper{}, newMockObjectStore())
	svc.SetLeaderLock(nil, db)
	release, acquired, err := svc.tryAcquireRecordsLock(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, release)
	release()
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBackupMonthlyArchiveRuntime_LeaderLockFallsBackAfterRedisError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	lockID := hashAdvisoryLockID(backupRecordsLeaderLockKey)
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).WithArgs(lockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectExec(`SELECT pg_advisory_unlock\(\$1\)`).WithArgs(lockID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := newTestBackupService(newMockSettingRepo(), &mockDumper{}, newMockObjectStore())
	svc.SetLeaderLock(&fakeLeaderLockCache{acquireErr: context.DeadlineExceeded}, db)
	release, acquired, err := svc.tryAcquireRecordsLock(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	release()
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBackupMonthlyArchiveRuntime_LeaderLockHonorsContextTimeout(t *testing.T) {
	svc := newTestBackupService(newMockSettingRepo(), &mockDumper{}, newMockObjectStore())
	svc.SetLeaderLock(blockingBackupLeaderLockCache{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := svc.saveRecord(ctx, &BackupRecord{ID: "lock-timeout", Status: "completed"})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
