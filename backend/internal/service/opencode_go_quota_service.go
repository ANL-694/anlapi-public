package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"anlapi/internal/config"
	infraerrors "anlapi/internal/pkg/errors"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const (
	openCodeGoQuotaTimeout = 15 * time.Second
	openCodeGoQuotaMaxBody = 256 * 1024

	openCodeGoUsageUpdatedAt     = "opencode_go_usage_updated_at"
	openCodeGoRollingUsedPercent = "opencode_go_5h_used_percent"
	openCodeGoRollingResetAt     = "opencode_go_5h_reset_at"
	openCodeGoWeeklyUsedPercent  = "opencode_go_weekly_used_percent"
	openCodeGoWeeklyResetAt      = "opencode_go_weekly_reset_at"
	openCodeGoMonthlyUsedPercent = "opencode_go_monthly_used_percent"
	openCodeGoMonthlyResetAt     = "opencode_go_monthly_reset_at"
)

// OpenCodeGoQuotaResult is the sanitized admin-facing projection of /usage.
// Credentials, upstream headers and raw response bodies are intentionally not
// included in this type.
type OpenCodeGoQuotaResult struct {
	AccountID    int64                `json:"account_id"`
	Plan         string               `json:"plan"`
	Data         *OpenCodeGoUsageData `json:"data,omitempty"`
	FetchedAt    time.Time            `json:"fetched_at"`
	Persisted    bool                 `json:"persisted"`
	CredentialOK bool                 `json:"credential_ok"`
	StatusCode   int                  `json:"status_code,omitempty"`
	Error        string               `json:"error,omitempty"`
}

// OpenCodeGoQuotaService queries and persists subscription usage windows for
// OpenCode Go accounts. Query calls for the same account share one upstream
// request through singleflight.
type OpenCodeGoQuotaService struct {
	accountRepo  AccountRepository
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream
	cfg          *config.Config
	flight       singleflight.Group
}

func NewOpenCodeGoQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *OpenCodeGoQuotaService {
	return &OpenCodeGoQuotaService{
		accountRepo:  accountRepo,
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
		cfg:          cfg,
	}
}

// QueryUsage fetches and persists the latest OpenCode Go usage snapshot.
func (s *OpenCodeGoQuotaService) QueryUsage(ctx context.Context, accountID int64) (*OpenCodeGoQuotaResult, error) {
	if s == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "OPENCODE_GO_QUOTA_NOT_CONFIGURED", "OpenCode Go quota service is not configured")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return nil, infraerrors.New(http.StatusNotFound, "OPENCODE_GO_QUOTA_ACCOUNT_NOT_FOUND", "OpenCode Go account not found")
	}
	if err := validateOpenCodeGoQuotaAccount(account); err != nil {
		return nil, err
	}

	key := "opencode_go_quota:" + strconv.FormatInt(account.ID, 10)
	resultCh := s.flight.DoChan(key, func() (any, error) {
		probeCtx, cancel := context.WithTimeout(context.Background(), openCodeGoQuotaTimeout)
		defer cancel()
		return s.queryUsageForAccount(probeCtx, account)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case flightResult := <-resultCh:
		if flightResult.Err != nil {
			return nil, flightResult.Err
		}
		result, ok := flightResult.Val.(*OpenCodeGoQuotaResult)
		if !ok || result == nil {
			return nil, infraerrors.New(http.StatusInternalServerError, "OPENCODE_GO_QUOTA_RESULT_INVALID", "invalid OpenCode Go quota result")
		}
		copy := *result
		return &copy, nil
	}
}

func validateOpenCodeGoQuotaAccount(account *Account) error {
	if account == nil {
		return infraerrors.New(http.StatusNotFound, "OPENCODE_GO_QUOTA_ACCOUNT_NOT_FOUND", "OpenCode Go account not found")
	}
	if !account.IsOpenCodeGo() {
		return infraerrors.New(http.StatusBadRequest, "OPENCODE_GO_QUOTA_INVALID_PLATFORM", "account is not an OpenCode Go account")
	}
	if !account.IsOpenCodeGoPlan() {
		return infraerrors.New(http.StatusBadRequest, "OPENCODE_GO_QUOTA_NOT_GO_PLAN", "Zen accounts do not expose OpenCode Go subscription windows")
	}
	if account.Type != AccountTypeAPIKey {
		return infraerrors.New(http.StatusBadRequest, "OPENCODE_GO_QUOTA_INVALID_TYPE", "OpenCode Go quota requires an API key account")
	}
	if strings.TrimSpace(account.GetOpenAIApiKey()) == "" {
		return infraerrors.New(http.StatusBadRequest, "OPENCODE_GO_QUOTA_KEY_MISSING", "OpenCode Go API key is missing")
	}
	return nil
}

func (s *OpenCodeGoQuotaService) queryUsageForAccount(ctx context.Context, account *Account) (*OpenCodeGoQuotaResult, error) {
	baseURL := account.GetOpenAIBaseURL()
	validated, err := (&OpenAIGatewayService{cfg: s.cfg}).validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, infraerrors.New(http.StatusForbidden, "OPENCODE_GO_QUOTA_URL_REJECTED", "OpenCode Go quota URL rejected")
	}
	targetURL := OpenCodeGoQuotaURL(validated)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "OPENCODE_GO_QUOTA_REQUEST_BUILD_FAILED", "failed to build quota request")
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(account.GetOpenAIApiKey()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-OpenCode-Session", uuid.NewString())

	proxyURL := s.resolveProxyURL(ctx, account)
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENCODE_GO_QUOTA_REQUEST_FAILED", "OpenCode Go quota request failed")
	}
	if resp == nil {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENCODE_GO_QUOTA_EMPTY_RESPONSE", "OpenCode Go quota returned no response")
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, openCodeGoQuotaMaxBody))
	if readErr != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENCODE_GO_QUOTA_READ_FAILED", "failed to read OpenCode Go quota response")
	}
	now := time.Now().UTC()
	result := &OpenCodeGoQuotaResult{
		AccountID:    account.ID,
		Plan:         AccountModeGo,
		FetchedAt:    now,
		StatusCode:   resp.StatusCode,
		CredentialOK: resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden,
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		result.Error = fmt.Sprintf("upstream returned HTTP %d", resp.StatusCode)
		return result, infraerrors.Newf(mapUpstreamStatus(resp.StatusCode), "OPENCODE_GO_QUOTA_UPSTREAM_ERROR", "OpenCode Go quota returned HTTP %d", resp.StatusCode)
	}
	usage, err := ParseOpenCodeGoUsageResponse(body)
	if err != nil {
		result.Error = "invalid usage response"
		return result, infraerrors.New(http.StatusBadGateway, "OPENCODE_GO_QUOTA_RESPONSE_INVALID", "OpenCode Go quota response is invalid")
	}
	result.Data = usage
	updates := buildOpenCodeGoQuotaExtraUpdates(usage, now)
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		result.Error = "quota snapshot persistence failed"
		return result, infraerrors.New(http.StatusInternalServerError, "OPENCODE_GO_QUOTA_PERSIST_FAILED", "failed to persist OpenCode Go quota snapshot")
	}
	result.Persisted = true
	return result, nil
}

func (s *OpenCodeGoQuotaService) resolveProxyURL(ctx context.Context, account *Account) string {
	if account == nil || account.ProxyID == nil {
		return ""
	}
	if account.Proxy != nil {
		return account.Proxy.URL()
	}
	if s.proxyRepo != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
		if err == nil && proxy != nil {
			account.Proxy = proxy
			return proxy.URL()
		}
	}
	return ""
}

func buildOpenCodeGoQuotaExtraUpdates(usage *OpenCodeGoUsageData, now time.Time) map[string]any {
	updates := map[string]any{openCodeGoUsageUpdatedAt: now.UTC().Format(time.RFC3339)}
	if usage == nil {
		return updates
	}
	setWindow := func(window OpenCodeGoUsageWindow, percentKey, resetKey string) {
		if window.Status != OpenCodeGoUsageStatusOK {
			return
		}
		updates[percentKey] = window.Percent
		if !window.ResetsAt.IsZero() {
			updates[resetKey] = window.ResetsAt.UTC().Format(time.RFC3339)
		}
	}
	setWindow(usage.Rolling, openCodeGoRollingUsedPercent, openCodeGoRollingResetAt)
	setWindow(usage.Weekly, openCodeGoWeeklyUsedPercent, openCodeGoWeeklyResetAt)
	setWindow(usage.Monthly, openCodeGoMonthlyUsedPercent, openCodeGoMonthlyResetAt)
	return updates
}
