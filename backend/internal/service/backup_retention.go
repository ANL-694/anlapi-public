package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	infraerrors "anlapi/internal/pkg/errors"
	"github.com/robfig/cron/v3"
)

const settingKeyBackupArchiveCheckpoint = "backup_monthly_archive_checkpoint"

// validateBackupRetention validates the persisted schedule contract without
// touching backup records or object storage.
func validateBackupRetention(cfg *BackupScheduleConfig) error {
	if cfg == nil {
		return infraerrors.BadRequest("INVALID_BACKUP_RETENTION", "backup retention config is required")
	}
	if cfg.RetainDays < 0 || cfg.RetainCount < 0 {
		return infraerrors.BadRequest("INVALID_BACKUP_RETENTION", "backup retention must not be negative")
	}
	archive := cfg.MonthlyArchive
	if archive == nil {
		return nil
	}
	if archive.RetainCount < 0 || len(archive.Days) > 31 {
		return infraerrors.BadRequest("INVALID_MONTHLY_ARCHIVE", "invalid archive retention or dates")
	}

	seen := make(map[int]struct{}, len(archive.Days))
	days := make([]int, 0, len(archive.Days))
	for _, day := range archive.Days {
		if day < 1 || day > 31 {
			return infraerrors.BadRequest("INVALID_MONTHLY_ARCHIVE", "archive dates must be between 1 and 31")
		}
		if _, ok := seen[day]; ok {
			continue
		}
		seen[day] = struct{}{}
		days = append(days, day)
	}
	sort.Ints(days)
	archive.Days = days
	if archive.Enabled && len(days) == 0 && !archive.IncludeMonthEnd {
		return infraerrors.BadRequest("INVALID_MONTHLY_ARCHIVE", "select at least one archive date")
	}
	return nil
}

// assignMonthlyArchive marks a completed scheduled backup for all archive dates
// since the last checkpoint. The caller must hold recordsMu. The checkpoint is
// returned separately so the record and checkpoint can be persisted together.
// This local implementation deliberately keeps the lock scope process-local;
// multi-instance Redis/advisory locking remains a separate production gate.
func (s *BackupService) assignMonthlyArchive(ctx context.Context, record *BackupRecord, schedule *BackupScheduleConfig) (string, error) {
	if record == nil || record.Status != "completed" || record.TriggeredBy != "scheduled" || record.MonthlyArchive != nil || schedule == nil || schedule.MonthlyArchive == nil || !schedule.MonthlyArchive.Enabled {
		return "", nil
	}
	if err := validateBackupRetention(schedule); err != nil {
		return "", err
	}
	startedAt, err := time.Parse(time.RFC3339, record.StartedAt)
	if err != nil {
		return "", fmt.Errorf("parse archive backup start: %w", err)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parser.Parse(schedule.CronExpr)
	if err != nil {
		return "", fmt.Errorf("parse archive schedule: %w", err)
	}
	location := time.Local
	if parsed, ok := spec.(*cron.SpecSchedule); ok {
		location = parsed.Location
	}
	startedAt = startedAt.In(location)
	today := startedAt.Format(time.DateOnly)
	checkpoint, err := s.settingRepo.GetValue(ctx, settingKeyBackupArchiveCheckpoint)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return "", fmt.Errorf("load archive checkpoint: %w", err)
	}
	if checkpoint != "" {
		if _, err := time.Parse(time.DateOnly, checkpoint); err != nil {
			return "", infraerrors.InternalServer("BACKUP_ARCHIVE_CHECKPOINT_CORRUPT", "archive checkpoint data is corrupted")
		}
		if checkpoint >= today {
			return "", nil
		}
	}
	dates := monthlyArchiveDates(startedAt, schedule.MonthlyArchive)
	due := make([]string, 0, len(dates))
	for _, date := range dates {
		if date > checkpoint && date <= today {
			due = append(due, date)
		}
	}
	if len(due) > 0 {
		record.MonthlyArchive = &BackupMonthlyArchive{Dates: due, RetainCount: schedule.MonthlyArchive.RetainCount}
		record.ExpiresAt = ""
	}
	return today, nil
}

// monthlyArchiveDates returns unique, sorted dates for the month containing at.
// Dates beyond the month's length are clamped to the month end.
func monthlyArchiveDates(at time.Time, cfg *BackupMonthlyArchiveConfig) []string {
	if cfg == nil {
		return nil
	}
	lastDay := time.Date(at.Year(), at.Month()+1, 0, 0, 0, 0, 0, at.Location()).Day()
	unique := make(map[int]struct{}, len(cfg.Days)+1)
	for _, day := range cfg.Days {
		if day < 1 {
			continue
		}
		if day > lastDay {
			day = lastDay
		}
		unique[day] = struct{}{}
	}
	if cfg.IncludeMonthEnd {
		unique[lastDay] = struct{}{}
	}
	dates := make([]string, 0, len(unique))
	for day := range unique {
		dates = append(dates, time.Date(at.Year(), at.Month(), day, 0, 0, 0, 0, at.Location()).Format(time.DateOnly))
	}
	sort.Strings(dates)
	return dates
}

func backupStartedAfter(a, b BackupRecord) bool {
	left, leftErr := time.Parse(time.RFC3339, a.StartedAt)
	right, rightErr := time.Parse(time.RFC3339, b.StartedAt)
	if leftErr == nil && rightErr == nil && !left.Equal(right) {
		return left.After(right)
	}
	if a.StartedAt != b.StartedAt {
		return a.StartedAt > b.StartedAt
	}
	return a.ID > b.ID
}
