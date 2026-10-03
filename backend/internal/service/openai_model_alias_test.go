package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeKnownOpenAICodexModel_BareGPT56RoutesToSol(t *testing.T) {
	tests := map[string]string{
		"gpt-5.6":            "gpt-5.6-sol",
		"openai/gpt-5.6":     "gpt-5.6-sol",
		"gpt5.6":             "gpt-5.6-sol",
		"gpt-5.6-high":       "gpt-5.6-sol",
		"gpt-5.6-max":        "gpt-5.6-sol",
		"gpt-5.6-2026-07-09": "gpt-5.6-sol",
		"openai/gpt-5.6-max": "gpt-5.6-sol",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, expected, normalizeKnownOpenAICodexModel(input))
		})
	}
}

func TestUsageBillingModelCandidates_BareGPT56IncludesSol(t *testing.T) {
	require.Equal(t,
		[]string{"gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("gpt-5.6"),
	)
	require.Equal(t,
		[]string{"openai/gpt-5.6", "gpt-5.6", "gpt-5.6-sol"},
		usageBillingModelCandidates("openai/gpt-5.6"),
	)
}

func TestNormalizeKnownOpenAICodexModel_GPT6SolLuna(t *testing.T) {
	tests := map[string]string{
		"gpt-6-sol":                "gpt-6-sol",
		"openai/gpt-6-luna-high":   "gpt-6-luna",
		"gpt-6-sol-openai-compact": "gpt-6-sol",
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			if got := normalizeKnownOpenAICodexModel(input); got != expected {
				t.Fatalf("normalizeKnownOpenAICodexModel(%q) = %q, want %q", input, got, expected)
			}
		})
	}
}

func TestNormalizeOpenAIReasoningEffortForModel_GPT6KeepsOfficialLevels(t *testing.T) {
	if got := normalizeOpenAIReasoningEffortForModel("none", "gpt-6-sol"); got != "none" {
		t.Fatalf("none normalized to %q", got)
	}
	if got := normalizeOpenAIReasoningEffortForModel("max", "gpt-6-luna"); got != "max" {
		t.Fatalf("max normalized to %q", got)
	}
}
