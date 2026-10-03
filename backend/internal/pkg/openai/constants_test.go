package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsPreferConcreteGPT56SolForAccountTests(t *testing.T) {
	require.NotEmpty(t, DefaultModels)
	require.Equal(t, "gpt-5.6-sol", DefaultModels[0].ID)
}

func TestGPT6SolLunaModelIdentity(t *testing.T) {
	for _, model := range []string{"gpt-6-sol", "gpt-6-luna"} {
		require.Contains(t, DefaultModelIDs(), model)
		require.True(t, IsGPT6SolOrLunaModelSpelling(model))
	}
	require.True(t, IsGPT6SolOrLunaModelSpelling("openai/gpt-6-sol-high"))
	require.False(t, IsGPT6SolOrLunaModelSpelling("gpt-6-astra"))
	require.False(t, IsGPT6SolOrLunaModelSpelling("gpt-6-solitude"))
	require.False(t, IsGPT6SolOrLunaModelSpelling("gpt-6-luna-preview"))
}

func TestGPT61SolModelIdentity(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-6.1-sol")
	for _, model := range []string{"gpt-6.1-sol", "openai/gpt-6.1-sol-high", "gpt-6.1-sol-openai-compact"} {
		require.True(t, IsGPT61SolModelSpelling(model), model)
	}
	require.False(t, IsGPT61SolModelSpelling("gpt-6.1-sol-preview"))
}
