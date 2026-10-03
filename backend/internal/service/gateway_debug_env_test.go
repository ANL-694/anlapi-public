package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayDebugEnvironmentUsesAnlapiNames(t *testing.T) {
	oldBodyPath := filepath.Join(t.TempDir(), "legacy-debug.log")
	newBodyPath := filepath.Join(t.TempDir(), "anlapi-debug.log")
	t.Setenv("ANLAPI_DEBUG_GATEWAY_BODY", oldBodyPath)
	t.Setenv("ANLAPI_DEBUG_MODEL_ROUTING", "true")
	t.Setenv("ANLAPI_DEBUG_CLAUDE_MIMIC", "true")
	t.Setenv(debugGatewayBodyEnv, "")
	t.Setenv(debugModelRoutingEnv, "")
	t.Setenv(debugClaudeMimicEnv, "")

	legacyOnly := NewGatewayService()
	require.False(t, legacyOnly.debugModelRoutingEnabled())
	require.False(t, legacyOnly.debugClaudeMimicEnabled())
	require.Nil(t, legacyOnly.debugGatewayBodyFile.Load())
	_, err := os.Stat(oldBodyPath)
	require.ErrorIs(t, err, os.ErrNotExist)

	t.Setenv(debugModelRoutingEnv, "true")
	t.Setenv(debugClaudeMimicEnv, "true")
	t.Setenv(debugGatewayBodyEnv, newBodyPath)
	current := NewGatewayService()
	require.True(t, current.debugModelRoutingEnabled())
	require.True(t, current.debugClaudeMimicEnabled())
	debugFile := current.debugGatewayBodyFile.Swap(nil)
	require.NotNil(t, debugFile)
	require.NoError(t, debugFile.Close())
	_, err = os.Stat(newBodyPath)
	require.NoError(t, err)
}

func TestParseDebugEnvBool(t *testing.T) {
	t.Run("empty is false", func(t *testing.T) {
		if parseDebugEnvBool("") {
			t.Fatalf("expected false for empty string")
		}
	})

	t.Run("true-like values", func(t *testing.T) {
		for _, value := range []string{"1", "true", "TRUE", "yes", "on"} {
			t.Run(value, func(t *testing.T) {
				if !parseDebugEnvBool(value) {
					t.Fatalf("expected true for %q", value)
				}
			})
		}
	})

	t.Run("false-like values", func(t *testing.T) {
		for _, value := range []string{"0", "false", "off", "debug"} {
			t.Run(value, func(t *testing.T) {
				if parseDebugEnvBool(value) {
					t.Fatalf("expected false for %q", value)
				}
			})
		}
	})
}
