//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	infraerrors "anlapi/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestBackupRetention_ValidationAndRoundTrip(t *testing.T) {
	for _, cfg := range []BackupScheduleConfig{
		{RetainDays: -1},
		{RetainCount: -1},
		{MonthlyArchive: &BackupMonthlyArchiveConfig{Enabled: true}},
		{MonthlyArchive: &BackupMonthlyArchiveConfig{Days: []int{0}}},
		{MonthlyArchive: &BackupMonthlyArchiveConfig{Days: []int{32}}},
		{MonthlyArchive: &BackupMonthlyArchiveConfig{RetainCount: -1}},
	} {
		require.Error(t, validateBackupRetention(&cfg))
	}

	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	cfg := BackupScheduleConfig{
		CronExpr: "0 4 * * *",
		MonthlyArchive: &BackupMonthlyArchiveConfig{
			Enabled: true, Days: []int{15, 1, 15}, IncludeMonthEnd: true, RetainCount: 17,
		},
	}
	saved, err := svc.UpdateSchedule(context.Background(), cfg)
	require.NoError(t, err)
	require.Equal(t, []int{1, 15}, saved.MonthlyArchive.Days)
	loaded, err := svc.GetSchedule(context.Background())
	require.NoError(t, err)
	require.Equal(t, saved, loaded)
}

func TestBackupRetention_MonthEndAndLeapYears(t *testing.T) {
	for _, tc := range []struct {
		date string
		want string
	}{
		{date: "2026-02-01", want: "2026-02-28"},
		{date: "2028-02-01", want: "2028-02-29"},
		{date: "2026-04-01", want: "2026-04-30"},
	} {
		t.Run(tc.date, func(t *testing.T) {
			at, err := time.Parse(time.DateOnly, tc.date)
			require.NoError(t, err)
			cfg := &BackupMonthlyArchiveConfig{Days: []int{30, 31}, IncludeMonthEnd: true}
			require.Equal(t, []string{tc.want}, monthlyArchiveDates(at, cfg))
		})
	}
}

func TestBackupRetention_UpdateScheduleRejectsNegativeValues(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	_, err := svc.UpdateSchedule(context.Background(), BackupScheduleConfig{RetainDays: -1})
	require.Equal(t, "INVALID_BACKUP_RETENTION", infraerrors.Reason(err))
}
