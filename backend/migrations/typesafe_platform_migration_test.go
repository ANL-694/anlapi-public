package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeSafePlatformMigrationExtendsOnlyRoutingAndQuotaChecks(t *testing.T) {
	content, err := FS.ReadFile("243_add_typesafe_platform.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ALTER TABLE user_platform_quotas")
	require.Contains(t, sql, "ALTER TABLE composite_model_routes")
	require.Contains(t, sql, "'typesafe'")
	require.NotContains(t, sql, "ALTER TABLE channel_monitors")
	require.NotContains(t, sql, "ALTER TABLE channel_monitor_request_templates")
}
