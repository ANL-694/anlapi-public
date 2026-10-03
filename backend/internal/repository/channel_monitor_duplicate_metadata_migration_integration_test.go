//go:build integration

package repository

import (
	"context"
	"testing"

	dbent "anlapi/ent"
	"anlapi/ent/channelmonitor"
	"anlapi/internal/service"
	"anlapi/migrations"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorDuplicateMetadataHardCutMigration(t *testing.T) {
	content, err := migrations.FS.ReadFile("242_channel_monitor_duplicate_metadata_hard_cut.sql")
	require.NoError(t, err)
	migrationSQL := string(content)

	t.Run("renames legacy key", func(t *testing.T) {
		tx := testEntTx(t)
		ctx := dbent.NewTxContext(context.Background(), tx)
		client := tx.Client()
		monitor := createMonitorWithDuplicateMetadata(t, ctx, client, map[string]string{
			"legacy:duplicate_operation_id": "operation-digest",
			"X-Original":                    "keep",
		})

		_, err := client.ExecContext(ctx, migrationSQL)
		require.NoError(t, err)

		stored, err := client.ChannelMonitor.Get(ctx, monitor.ID)
		require.NoError(t, err)
		require.Equal(t, "operation-digest", stored.ExtraHeaders[service.ChannelMonitorDuplicateOperationIDMetadataKey])
		require.Equal(t, "keep", stored.ExtraHeaders["X-Original"])
		require.NotContains(t, stored.ExtraHeaders, "legacy:duplicate_operation_id")
	})

	t.Run("removes matching duplicate keys", func(t *testing.T) {
		tx := testEntTx(t)
		ctx := dbent.NewTxContext(context.Background(), tx)
		client := tx.Client()
		monitor := createMonitorWithDuplicateMetadata(t, ctx, client, map[string]string{
			"legacy:duplicate_operation_id":                       "same-digest",
			service.ChannelMonitorDuplicateOperationIDMetadataKey: "same-digest",
		})

		_, err := client.ExecContext(ctx, migrationSQL)
		require.NoError(t, err)

		stored, err := client.ChannelMonitor.Get(ctx, monitor.ID)
		require.NoError(t, err)
		require.Equal(t, "same-digest", stored.ExtraHeaders[service.ChannelMonitorDuplicateOperationIDMetadataKey])
		require.NotContains(t, stored.ExtraHeaders, "legacy:duplicate_operation_id")
	})

	t.Run("rejects conflicting values", func(t *testing.T) {
		ctx := context.Background()
		client := testEntClient(t)
		monitor := createMonitorWithDuplicateMetadata(t, ctx, client, map[string]string{
			"legacy:duplicate_operation_id":                       "old-digest",
			service.ChannelMonitorDuplicateOperationIDMetadataKey: "new-digest",
		})
		t.Cleanup(func() {
			require.NoError(t, client.ChannelMonitor.DeleteOneID(monitor.ID).Exec(ctx))
		})

		_, err := integrationDB.ExecContext(ctx, migrationSQL)
		require.ErrorContains(t, err, "conflicting old and new values")

		stored, err := client.ChannelMonitor.Get(ctx, monitor.ID)
		require.NoError(t, err)
		require.Equal(t, "old-digest", stored.ExtraHeaders["legacy:duplicate_operation_id"])
		require.Equal(t, "new-digest", stored.ExtraHeaders[service.ChannelMonitorDuplicateOperationIDMetadataKey])
	})
}

func createMonitorWithDuplicateMetadata(
	t *testing.T,
	ctx context.Context,
	client *dbent.Client,
	headers map[string]string,
) *dbent.ChannelMonitor {
	t.Helper()

	monitor, err := client.ChannelMonitor.Create().
		SetName("metadata-migration").
		SetProvider(channelmonitor.ProviderOpenai).
		SetAPIMode(service.MonitorAPIModeResponses).
		SetEndpoint("https://api.example.com").
		SetAPIKeyEncrypted("encrypted-key").
		SetPrimaryModel("gpt-5.4-mini").
		SetIntervalSeconds(60).
		SetCreatedBy(1).
		SetExtraHeaders(headers).
		Save(ctx)
	require.NoError(t, err)
	return monitor
}
