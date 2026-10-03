package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"anlapi/internal/pkg/httputil"
	"anlapi/internal/pkg/ip"
	"anlapi/internal/pkg/logger"
	"anlapi/internal/pkg/typesafe"
	middleware2 "anlapi/internal/server/middleware"
	"anlapi/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const systemOneOnlyPlatformMessage = "TypeSafe models are only available through POST /v1/systemone"

// rejectSystemOneOnlyPlatform prevents a TypeSafe group from entering a
// Claude/OpenAI protocol handler. TypeSafe has a native JSON contract only.
func rejectSystemOneOnlyPlatform(c *gin.Context, apiKey *service.APIKey, writeError func(*gin.Context, int, string, string)) bool {
	if c == nil || apiKey == nil || writeError == nil {
		return false
	}
	if _, forced := middleware2.GetForcePlatformFromContext(c); forced {
		return false
	}
	if effectiveAPIKeyPlatform(c, apiKey) != service.PlatformTypeSafe {
		return false
	}
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
	writeError(c, http.StatusNotFound, "not_found_error", systemOneOnlyPlatformMessage)
	return true
}

// SystemOne handles TypeSafe's native, non-streaming System One protocol.
func (h *GatewayHandler) SystemOne(c *gin.Context) {
	started := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.Group == nil {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	if h == nil || h.gatewayService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Gateway service is not configured")
		return
	}
	reqLog := requestLogger(c, "handler.gateway.systemone",
		zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.Any("group_id", apiKey.GroupID))

	body, err := httputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, isMax := extractMaxBytesError(err); isMax {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	model, err := typesafe.ValidateSystemOneRequest(body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	ensureCompositeTargetPlatform(c, apiKey, model)
	if apiKey.Group.Platform != service.PlatformTypeSafe &&
		(apiKey.Group.Platform != service.PlatformComposite || !compositeTargetPlatformAllowed(c, apiKey, model, service.PlatformTypeSafe)) {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "System One is only available for TypeSafe and compatible Composite groups")
		return
	}
	if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolTypeSafeSystemOne, model, body); decision != nil && !decision.AllowNextStage {
		h.anthropicSecurityAuditError(c, decision)
		return
	}

	setOpsRequestContext(c, model, false, body)
	setOpsEndpointContext(c, "", int16(service.RequestTypeSync))
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(started).Milliseconds())
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	streamStarted := false
	if h.concurrencyHelper == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Gateway concurrency service is not configured")
		return
	}
	userRelease, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		h.handleConcurrencyError(c, err, "user", false)
		return
	}
	userRelease = wrapReleaseOnDone(c.Request.Context(), userRelease)
	if userRelease != nil {
		defer userRelease()
	}
	if h.billingCacheService == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Billing service is not configured")
		return
	}
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", formatRetryAfter(retryAfter))
		}
		h.errorResponse(c, status, code, message)
		return
	}

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, model)
	fs := NewFailoverState(h.maxAccountSwitches, false)
	for {
		if failoverClientGone(c) {
			return
		}
		selection, selectErr := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, "", model, fs.FailedAccountIDs, "", subject.UserID)
		if selectErr != nil || selection == nil || selection.Account == nil {
			if len(fs.FailedAccountIDs) == 0 {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available TypeSafe accounts")
				return
			}
			switch fs.HandleSelectionExhausted(c.Request.Context()) {
			case FailoverContinue:
				continue
			case FailoverCanceled:
				return
			default:
				if fs.LastFailoverErr != nil {
					h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformTypeSafe, false)
				} else {
					h.handleFailoverExhaustedSimple(c, http.StatusBadGateway, false)
				}
				return
			}
		}

		account := selection.Account
		accountRelease := selection.ReleaseFunc
		if !selection.Acquired {
			if selection.WaitPlan == nil || h.concurrencyHelper == nil {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available TypeSafe accounts")
				return
			}
			accountRelease, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, false, &streamStarted)
			if err != nil {
				h.handleConcurrencyError(c, err, "account", false)
				return
			}
		}
		accountRelease = wrapReleaseOnDone(c.Request.Context(), accountRelease)
		setOpsSelectedAccount(c, account.ID, account.Platform)
		setOpsEndpointContext(c, model, int16(service.RequestTypeSync))
		result, forwardErr := h.gatewayService.ForwardSystemOne(c.Request.Context(), c, account, body)
		if accountRelease != nil {
			accountRelease()
		}
		if forwardErr != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(forwardErr, &failoverErr) {
				switch fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, account.GetPoolModeRetryCount(), failoverErr) {
				case FailoverContinue:
					continue
				case FailoverCanceled:
					return
				default:
					h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformTypeSafe, false)
					return
				}
			}
			var requestErr *service.SystemOneUpstreamError
			if errors.As(forwardErr, &requestErr) {
				status := requestErr.StatusCode
				if !service.IsSystemOneRequestErrorStatus(status) {
					status = http.StatusBadGateway
				}
				h.errorResponse(c, status, "upstream_error", "TypeSafe rejected the request")
				return
			}
			if errors.Is(forwardErr, typesafe.ErrSystemOneResponseTooLarge) {
				h.errorResponse(c, http.StatusBadGateway, "upstream_error", "TypeSafe response exceeds the gateway size limit")
				return
			}
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "TypeSafe upstream request failed")
			return
		}

		c.Data(result.StatusCode, result.ContentType, result.Body)
		h.recordSystemOneUsage(c, apiKey, account, subscription, channelMapping, model, body, result, subject.UserID)
		return
	}
}

func (h *GatewayHandler) recordSystemOneUsage(c *gin.Context, apiKey *service.APIKey, account *service.Account, subscription *service.UserSubscription, mapping service.ChannelMappingResult, model string, body []byte, result *service.SystemOneForwardResult, userID int64) {
	if h == nil || h.gatewayService == nil || result == nil {
		return
	}
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	requestHash := service.HashUsageRequestPayload(body)
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	sessionID := service.ExtractClientSessionID(c)
	usageFields := mapping.ToUsageFields(clientRequestedModel(c, model), result.UpstreamModel)
	h.submitUsageRecordTask(func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
			Result: &result.ForwardResult, APIKey: apiKey, User: apiKey.User, Account: account,
			Subscription: subscription, InboundEndpoint: inboundEndpoint, UpstreamEndpoint: upstreamEndpoint,
			UserAgent: userAgent, IPAddress: clientIP, SessionID: sessionID,
			RequestPayloadHash: requestHash, APIKeyService: h.apiKeyService, QuotaPlatform: quotaPlatform,
			ChannelUsageFields: usageFields,
		}); err != nil {
			logger.L().With(zap.String("component", "handler.gateway.systemone"), zap.Int64("user_id", userID), zap.Int64("api_key_id", apiKey.ID), zap.Int64("account_id", account.ID)).Error("systemone.record_usage_failed", zap.Error(err))
		}
	})
}

func formatRetryAfter(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	return strconv.Itoa(seconds)
}
