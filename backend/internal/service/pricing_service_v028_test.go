package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPricingServiceV028ModelsUseExactStaticFallbacks(t *testing.T) {
	svc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{}}

	sol := svc.matchOpenAIModel("openai/gpt-6-sol-high")
	require.NotNil(t, sol)
	require.InDelta(t, 2e-6, sol.InputCostPerToken, 1e-15)
	require.InDelta(t, 10e-6, sol.OutputCostPerToken, 1e-15)
	require.Equal(t, 272000, sol.LongContextInputTokenThreshold)

	luna := svc.matchOpenAIModel("gpt-6-luna")
	require.NotNil(t, luna)
	require.InDelta(t, 0.1e-6, luna.InputCostPerToken, 1e-15)
	require.InDelta(t, 0.5e-6, luna.OutputCostPerToken, 1e-15)
}

func TestPricingServiceV028Opus55DoesNotUseOpus5FamilyFallback(t *testing.T) {
	svc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"claude-opus-5": {InputCostPerToken: 99},
	}}

	pricing := svc.matchByModelFamily("claude-opus-5-5")
	require.NotNil(t, pricing)
	require.InDelta(t, 4e-6, pricing.InputCostPerToken, 1e-15)
	require.InDelta(t, 20e-6, pricing.OutputCostPerToken, 1e-15)
}
