package service

import (
	"testing"

	"anlapi/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingServiceV028OfficialFixtureMapsSupportedPricing(t *testing.T) {
	pricingData, err := (&PricingService{}).parsePricingData([]byte(officialV028ModelPricingFixture))
	require.NoError(t, err)

	svc := NewBillingService(&config.Config{}, &PricingService{pricingData: pricingData})

	for _, tc := range []struct {
		model          string
		input          float64
		inputPriority  float64
		output         float64
		outputPriority float64
		cache          float64
		cachePriority  float64
		read           float64
		readPriority   float64
	}{
		{model: "gpt-6-sol", input: 2e-6, inputPriority: 4e-6, output: 10e-6, outputPriority: 20e-6, cache: 2.5e-6, cachePriority: 5e-6, read: 0.2e-6, readPriority: 0.4e-6},
		{model: "gpt-6-luna", input: 0.1e-6, inputPriority: 0.2e-6, output: 0.5e-6, outputPriority: 1e-6, cache: 0.125e-6, cachePriority: 0.25e-6, read: 0.01e-6, readPriority: 0.02e-6},
		{model: "claude-opus-5-5", input: 4e-6, inputPriority: 8e-6, output: 20e-6, outputPriority: 40e-6, cache: 5e-6, cachePriority: 10e-6, read: 0.2e-6, readPriority: 0.4e-6},
	} {
		pricing, err := svc.GetModelPricing(tc.model)
		require.NoError(t, err)
		require.InDelta(t, tc.input, pricing.InputPricePerToken, 1e-15, tc.model)
		require.InDelta(t, tc.inputPriority, pricing.InputPricePerTokenPriority, 1e-15, tc.model)
		require.InDelta(t, tc.output, pricing.OutputPricePerToken, 1e-15, tc.model)
		require.InDelta(t, tc.outputPriority, pricing.OutputPricePerTokenPriority, 1e-15, tc.model)
		require.InDelta(t, tc.cache, pricing.CacheCreationPricePerToken, 1e-15, tc.model)
		require.InDelta(t, tc.cachePriority, pricing.CacheCreationPricePerTokenPriority, 1e-15, tc.model)
		require.InDelta(t, tc.read, pricing.CacheReadPricePerToken, 1e-15, tc.model)
		require.InDelta(t, tc.readPriority, pricing.CacheReadPricePerTokenPriority, 1e-15, tc.model)
	}

	opus, err := svc.GetModelPricing("claude-opus-5-5")
	require.NoError(t, err)
	require.True(t, opus.SupportsCacheBreakdown)
	require.NotNil(t, opus.FastMultiplier)
	require.InDelta(t, 2.0, *opus.FastMultiplier, 1e-15)
	require.InDelta(t, 8e-6, opus.CacheCreation1hPrice, 1e-15)
	require.InDelta(t, 8e-6, opus.InputPricePerTokenPriority, 1e-15)
}

func TestBillingServiceV028Opus55FastMultiplierMatchesFallback(t *testing.T) {
	pricingData, err := (&PricingService{}).parsePricingData([]byte(officialV028ModelPricingFixture))
	require.NoError(t, err)

	dynamicSvc := NewBillingService(&config.Config{}, &PricingService{pricingData: pricingData})
	dynamic, err := dynamicSvc.GetModelPricing("claude-opus-5-5")
	require.NoError(t, err)
	require.NotNil(t, dynamic.FastMultiplier)

	fallbackSvc := NewBillingService(&config.Config{}, nil)
	fallback, err := fallbackSvc.GetModelPricing("claude-opus-5-5")
	require.NoError(t, err)
	require.NotNil(t, fallback.FastMultiplier)
	require.InDelta(t, *fallback.FastMultiplier, *dynamic.FastMultiplier, 1e-15)
}

func TestBillingServiceFastAndFlexMultipliersScaleCacheBreakdown(t *testing.T) {
	fastMultiplier := 2.0
	flexMultiplier := 0.5
	// Fast/Flex are billing-layer fields, so build ModelPricing directly for the
	// cache-tier assertion without pretending the remote LiteLLM schema already
	// supplies these values.
	svc := NewBillingService(&config.Config{}, &PricingService{})
	pricing := &ModelPricing{
		InputPricePerToken:         1e-6,
		OutputPricePerToken:        2e-6,
		CacheCreationPricePerToken: 3e-6,
		CacheReadPricePerToken:     4e-6,
		FastMultiplier:             &fastMultiplier,
		FlexMultiplier:             &flexMultiplier,
		SupportsCacheBreakdown:     true,
		CacheCreation5mPrice:       3e-6,
		CacheCreation1hPrice:       6e-6,
	}
	svc.fallbackPrices["gpt-6-sol"] = pricing

	tokens := UsageTokens{
		InputTokens:           100,
		OutputTokens:          20,
		CacheCreationTokens:   15,
		CacheCreation5mTokens: 10,
		CacheCreation1hTokens: 5,
		CacheReadTokens:       10,
	}
	base, err := svc.CalculateCost("gpt-6-sol", tokens, 1)
	require.NoError(t, err)
	fast, err := svc.CalculateCostWithServiceTier("gpt-6-sol", tokens, 1, "fast")
	require.NoError(t, err)
	flex, err := svc.CalculateCostWithServiceTier("gpt-6-sol", tokens, 1, "flex")
	require.NoError(t, err)

	require.InDelta(t, base.InputCost*2, fast.InputCost, 1e-15)
	require.InDelta(t, base.OutputCost*2, fast.OutputCost, 1e-15)
	require.InDelta(t, base.CacheCreationCost*2, fast.CacheCreationCost, 1e-15)
	require.InDelta(t, base.CacheReadCost*2, fast.CacheReadCost, 1e-15)
	require.InDelta(t, base.TotalCost*0.5, flex.TotalCost, 1e-15)
}
