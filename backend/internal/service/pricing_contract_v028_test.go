package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// This fixture is copied from official v0.2.8 commit c4c2e660 and intentionally
// keeps fields that LiteLLMModelPricing supports alongside fields it does not.
const officialV028ModelPricingFixture = `{
  "gpt-6-sol": {
    "cache_creation_input_token_cost": 2.5e-6,
    "cache_creation_input_token_cost_batches": 1.25e-6,
    "cache_creation_input_token_cost_flex": 1.25e-6,
    "cache_creation_input_token_cost_priority": 5e-6,
    "cache_read_input_token_cost": 2e-7,
    "cache_read_input_token_cost_batches": 1e-7,
    "cache_read_input_token_cost_flex": 1e-7,
    "cache_read_input_token_cost_priority": 4e-7,
    "input_cost_per_token": 2e-6,
    "input_cost_per_token_batches": 1e-6,
    "input_cost_per_token_flex": 1e-6,
    "input_cost_per_token_priority": 4e-6,
    "long_context_input_token_threshold": 272000,
    "long_context_input_cost_multiplier": 2.0,
    "long_context_output_cost_multiplier": 1.5,
    "litellm_provider": "openai",
    "max_input_tokens": 922000,
    "max_output_tokens": 128000,
    "mode": "chat",
    "output_cost_per_token": 1e-5,
    "output_cost_per_token_batches": 5e-6,
    "output_cost_per_token_flex": 5e-6,
    "output_cost_per_token_priority": 2e-5,
    "supports_prompt_caching": true,
    "supports_service_tier": true,
    "context_window": 1050000
  },
  "gpt-6-luna": {
    "cache_creation_input_token_cost": 1.25e-7,
    "cache_creation_input_token_cost_batches": 6.25e-8,
    "cache_creation_input_token_cost_flex": 6.25e-8,
    "cache_creation_input_token_cost_priority": 2.5e-7,
    "cache_read_input_token_cost": 1e-8,
    "cache_read_input_token_cost_batches": 5e-9,
    "cache_read_input_token_cost_flex": 5e-9,
    "cache_read_input_token_cost_priority": 2e-8,
    "input_cost_per_token": 1e-7,
    "input_cost_per_token_batches": 5e-8,
    "input_cost_per_token_flex": 5e-8,
    "input_cost_per_token_priority": 2e-7,
    "long_context_input_token_threshold": 272000,
    "long_context_input_cost_multiplier": 2.0,
    "long_context_output_cost_multiplier": 1.5,
    "litellm_provider": "openai",
    "max_input_tokens": 922000,
    "max_output_tokens": 128000,
    "mode": "chat",
    "output_cost_per_token": 5e-7,
    "output_cost_per_token_batches": 2.5e-7,
    "output_cost_per_token_flex": 2.5e-7,
    "output_cost_per_token_priority": 1e-6,
    "supports_prompt_caching": true,
    "supports_service_tier": true,
    "context_window": 1050000
  },
  "claude-opus-5-5": {
    "cache_creation_input_token_cost": 5e-6,
    "cache_creation_input_token_cost_above_1hr": 8e-6,
    "cache_creation_input_token_cost_priority": 1e-5,
    "cache_read_input_token_cost": 2e-7,
    "cache_read_input_token_cost_priority": 4e-7,
    "input_cost_per_token": 4e-6,
    "input_cost_per_token_priority": 8e-6,
    "litellm_provider": "anthropic",
    "max_input_tokens": 1000000,
    "max_output_tokens": 128000,
    "max_tokens": 128000,
    "mode": "chat",
    "output_cost_per_token": 2e-5,
    "input_cost_per_token_batches": 2e-6,
    "output_cost_per_token_batches": 1e-5,
    "output_cost_per_token_priority": 4e-5,
    "provider_specific_entry": {"fast": 2.0},
    "supports_adaptive_thinking": true,
    "supports_prompt_caching": true
  }
}`

func assertV028SupportedPricingFields(t *testing.T, expected, actual *LiteLLMModelPricing, compareServiceTier bool) {
	t.Helper()
	require.InDelta(t, expected.InputCostPerToken, actual.InputCostPerToken, 1e-15)
	require.InDelta(t, expected.InputCostPerTokenPriority, actual.InputCostPerTokenPriority, 1e-15)
	require.InDelta(t, expected.OutputCostPerToken, actual.OutputCostPerToken, 1e-15)
	require.InDelta(t, expected.OutputCostPerTokenPriority, actual.OutputCostPerTokenPriority, 1e-15)
	require.InDelta(t, expected.CacheCreationInputTokenCost, actual.CacheCreationInputTokenCost, 1e-15)
	require.InDelta(t, expected.CacheCreationInputTokenCostPriority, actual.CacheCreationInputTokenCostPriority, 1e-15)
	require.InDelta(t, expected.CacheCreationInputTokenCostAbove1hr, actual.CacheCreationInputTokenCostAbove1hr, 1e-15)
	require.InDelta(t, expected.CacheReadInputTokenCost, actual.CacheReadInputTokenCost, 1e-15)
	require.InDelta(t, expected.CacheReadInputTokenCostPriority, actual.CacheReadInputTokenCostPriority, 1e-15)
	require.Equal(t, expected.LongContextInputTokenThreshold, actual.LongContextInputTokenThreshold)
	require.InDelta(t, expected.LongContextInputCostMultiplier, actual.LongContextInputCostMultiplier, 1e-15)
	require.InDelta(t, expected.LongContextOutputCostMultiplier, actual.LongContextOutputCostMultiplier, 1e-15)
	if compareServiceTier {
		require.Equal(t, expected.SupportsServiceTier, actual.SupportsServiceTier)
	}
	require.Equal(t, expected.LiteLLMProvider, actual.LiteLLMProvider)
	require.Equal(t, expected.Mode, actual.Mode)
	require.Equal(t, expected.SupportsPromptCaching, actual.SupportsPromptCaching)
}

func TestPricingServiceV028OfficialFixtureMatchesSupportedFallbackFields(t *testing.T) {
	svc := &PricingService{}
	parsed, err := svc.parsePricingData([]byte(officialV028ModelPricingFixture))
	require.NoError(t, err)

	assertV028SupportedPricingFields(t, openAIGPT6SolFallbackPricing, parsed["gpt-6-sol"], true)
	assertV028SupportedPricingFields(t, openAIGPT6LunaFallbackPricing, parsed["gpt-6-luna"], true)
	assertV028SupportedPricingFields(t, claudeOpus55FallbackPricing, parsed["claude-opus-5-5"], false)
}

func TestPricingServiceV028Opus55PreservesOfficialServiceTierFallbackOverride(t *testing.T) {
	svc := &PricingService{}
	parsed, err := svc.parsePricingData([]byte(officialV028ModelPricingFixture))
	require.NoError(t, err)

	// The official resource omits this boolean, while official pricing_service.go
	// intentionally enables the priority fallback for Opus 5.5.
	require.False(t, parsed["claude-opus-5-5"].SupportsServiceTier)
	require.True(t, claudeOpus55FallbackPricing.SupportsServiceTier)
}

func TestPricingServiceV028OfficialFixturePreservesMetadataAndAlternateRates(t *testing.T) {
	svc := &PricingService{}
	parsed, err := svc.parsePricingData([]byte(officialV028ModelPricingFixture))
	require.NoError(t, err)

	sol := parsed["gpt-6-sol"]
	require.True(t, sol.CacheCreationInputTokenCostExplicit)
	require.Equal(t, 1050000, sol.ContextWindow)
	require.Equal(t, 922000, sol.MaxInputTokens)
	require.Equal(t, 128000, sol.MaxOutputTokens)
	require.InDelta(t, 1e-6, sol.InputCostPerTokenBatches, 1e-15)
	require.InDelta(t, 1e-6, sol.InputCostPerTokenFlex, 1e-15)
	require.InDelta(t, 1.25e-6, sol.CacheCreationInputTokenCostBatches, 1e-15)
	require.InDelta(t, 1.25e-6, sol.CacheCreationInputTokenCostFlex, 1e-15)

	opus := parsed["claude-opus-5-5"]
	require.True(t, opus.CacheCreationInputTokenCostExplicit)
	require.True(t, opus.SupportsAdaptiveThinking)
	require.Equal(t, 128000, opus.MaxTokens)
	require.InDelta(t, 2e-6, opus.InputCostPerTokenBatches, 1e-15)
	require.InDelta(t, 2.0, opus.ProviderSpecificEntry["fast"], 1e-15)
}

func TestPricingServiceV028OfficialFixtureAuditsUnsupportedFields(t *testing.T) {
	var raw map[string]map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(officialV028ModelPricingFixture), &raw))

	for _, model := range []string{"gpt-6-sol", "gpt-6-luna"} {
		var contextWindow, maxInput, maxOutput int
		require.NoError(t, json.Unmarshal(raw[model]["context_window"], &contextWindow))
		require.NoError(t, json.Unmarshal(raw[model]["max_input_tokens"], &maxInput))
		require.NoError(t, json.Unmarshal(raw[model]["max_output_tokens"], &maxOutput))
		require.Equal(t, 1050000, contextWindow)
		require.Equal(t, 922000, maxInput)
		require.Equal(t, 128000, maxOutput)
	}

	var maxInput, maxOutput int
	var adaptiveThinking bool
	var providerSpecific map[string]float64
	require.NoError(t, json.Unmarshal(raw["claude-opus-5-5"]["max_input_tokens"], &maxInput))
	require.NoError(t, json.Unmarshal(raw["claude-opus-5-5"]["max_output_tokens"], &maxOutput))
	require.NoError(t, json.Unmarshal(raw["claude-opus-5-5"]["supports_adaptive_thinking"], &adaptiveThinking))
	require.NoError(t, json.Unmarshal(raw["claude-opus-5-5"]["provider_specific_entry"], &providerSpecific))
	require.Equal(t, 1000000, maxInput)
	require.Equal(t, 128000, maxOutput)
	require.True(t, adaptiveThinking)
	require.Equal(t, 2.0, providerSpecific["fast"])
}
