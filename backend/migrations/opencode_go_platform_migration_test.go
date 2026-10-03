package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenCodeGoPlatformMigrationKeepsANLProviderSet(t *testing.T) {
	content, err := FS.ReadFile("209_opencode_go_platform.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "to_regclass('public.user_platform_quotas') IS NOT NULL")
	require.Contains(t, sql, "to_regclass('public.composite_model_routes') IS NOT NULL")
	require.Contains(t, sql, "to_regclass('public.channel_monitors') IS NOT NULL")
	require.Contains(t, sql, "to_regclass('public.channel_monitor_request_templates') IS NOT NULL")
	for _, provider := range []string{"'kimi'", "'zhipu'", "'deepseek'", "'minimax'", "'kiro'", "'opencode_go'"} {
		require.Contains(t, sql, provider)
	}
}
