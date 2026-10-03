package service

import (
	"testing"
	"time"
)

func TestBuildOpenCodeGoQuotaExtraUpdatesUsesOnlyValidWindows(t *testing.T) {
	now := time.Date(2026, time.September, 25, 1, 2, 3, 0, time.UTC)
	reset := now.Add(2 * time.Hour)
	updates := buildOpenCodeGoQuotaExtraUpdates(&OpenCodeGoUsageData{
		Rolling: OpenCodeGoUsageWindow{Status: OpenCodeGoUsageStatusOK, Percent: 12.5, ResetsAt: reset},
		Weekly:  OpenCodeGoUsageWindow{Status: "unknown", Percent: 80},
		Monthly: OpenCodeGoUsageWindow{Status: OpenCodeGoUsageStatusOK, Percent: 33.3},
	}, now)
	if got := updates[openCodeGoUsageUpdatedAt]; got != now.Format(time.RFC3339) {
		t.Fatalf("updated_at = %v", got)
	}
	if got := updates[openCodeGoRollingUsedPercent]; got != 12.5 {
		t.Fatalf("rolling percent = %v", got)
	}
	if got := updates[openCodeGoRollingResetAt]; got != reset.Format(time.RFC3339) {
		t.Fatalf("rolling reset = %v", got)
	}
	if _, ok := updates[openCodeGoWeeklyUsedPercent]; ok {
		t.Fatal("unknown weekly window must not overwrite the snapshot")
	}
	if got := updates[openCodeGoMonthlyUsedPercent]; got != 33.3 {
		t.Fatalf("monthly percent = %v", got)
	}
}

func TestValidateOpenCodeGoQuotaAccountIsStrict(t *testing.T) {
	valid := &Account{Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "redacted"}}
	if err := validateOpenCodeGoQuotaAccount(valid); err != nil {
		t.Fatalf("valid account rejected: %v", err)
	}
	zen := &Account{Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "redacted", "account_mode": AccountModeZen}}
	if err := validateOpenCodeGoQuotaAccount(zen); err == nil {
		t.Fatal("Zen account must not use Go quota windows")
	}
	missingKey := &Account{Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Credentials: map[string]any{}}
	if err := validateOpenCodeGoQuotaAccount(missingKey); err == nil {
		t.Fatal("account without API key must be rejected")
	}
}
