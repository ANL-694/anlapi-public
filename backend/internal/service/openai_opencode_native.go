package service

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// forwardOpenCodeChatViaNativeAnthropic reuses ANL's existing Anthropic
// conversion state machine for OpenCode models whose native protocol is
// Anthropic. The adapter keeps OpenAI billing/result semantics at the boundary.
func (s *OpenAIGatewayService) forwardOpenCodeChatViaNativeAnthropic(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	gateway := &GatewayService{
		cfg:                  s.cfg,
		httpUpstream:         s.httpUpstream,
		deferredService:      s.deferredService,
		rateLimitService:     s.rateLimitService,
		responseHeaderFilter: s.responseHeaderFilter,
		settingService:       s.settingService,
		identityService:      nil,
	}
	gateway.tlsFPProfileService = &TLSFingerprintProfileService{}
	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	forwardBody := body
	if mapped := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel); mapped != "" && mapped != originalModel {
		forwardBody = ReplaceModelInBody(body, mapped)
	}
	result, err := gateway.ForwardAsChatCompletions(ctx, c, account, forwardBody, nil)
	if result == nil {
		return nil, err
	}
	upstreamModel := result.UpstreamModel
	if upstreamModel == "" {
		upstreamModel = strings.TrimSpace(gjson.GetBytes(forwardBody, "model").String())
	}
	return &OpenAIForwardResult{
		RequestID:        result.RequestID,
		Usage:            OpenAIUsage{InputTokens: result.Usage.InputTokens, OutputTokens: result.Usage.OutputTokens, CacheCreationInputTokens: result.Usage.CacheCreationInputTokens, CacheReadInputTokens: result.Usage.CacheReadInputTokens},
		Model:            originalModel,
		BillingModel:     upstreamModel,
		UpstreamModel:    upstreamModel,
		UpstreamEndpoint: "/v1/messages",
		ReasoningEffort:  result.ReasoningEffort,
		Stream:           result.Stream,
		Duration:         result.Duration,
		FirstTokenMs:     result.FirstTokenMs,
		ClientDisconnect: result.ClientDisconnect,
	}, err
}
