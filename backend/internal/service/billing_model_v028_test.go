//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFallbackPricingIncludesV028Models(t *testing.T) {
	svc := newTestBillingService()

	tests := []struct {
		model  string
		input  float64
		output float64
	}{
		{model: "gpt-6-sol", input: 2e-6, output: 10e-6},
		{model: "openai/gpt-6-luna-high", input: 0.1e-6, output: 0.5e-6},
		{model: "claude-opus-5-5", input: 4e-6, output: 20e-6},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricing := svc.getFallbackPricing(tt.model)
			require.NotNil(t, pricing)
			require.InDelta(t, tt.input, pricing.InputPricePerToken, 1e-12)
			require.InDelta(t, tt.output, pricing.OutputPricePerToken, 1e-12)
		})
	}
}

func TestCalculateCostV028ModelsUsesIndependentFallbackRates(t *testing.T) {
	svc := newTestBillingService()

	cost, err := svc.CalculateCost("gpt-6-sol", UsageTokens{InputTokens: 1000, OutputTokens: 1000}, 1)
	require.NoError(t, err)
	require.InDelta(t, 0.012, cost.ActualCost, 1e-12)

	cost, err = svc.CalculateCost("claude-opus-5-5", UsageTokens{InputTokens: 1000, OutputTokens: 1000}, 1)
	require.NoError(t, err)
	require.InDelta(t, 0.024, cost.ActualCost, 1e-12)
}
