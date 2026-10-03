package repository

import (
	"context"
	"testing"

	dbent "anlapi/ent"
	"anlapi/internal/pkg/pagination"
	"anlapi/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepositoryListByUserIDSortByGroup(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "group-sort@test.com")
	createGroup := func(name string) *dbent.Group {
		g, err := client.Group.Create().SetName(name).Save(ctx)
		require.NoError(t, err)
		return g
	}
	zulu := createGroup("Zulu")
	alpha := createGroup("Alpha")
	createKey := func(name string, groupID *int64) int64 {
		key := &service.APIKey{UserID: user.ID, Key: "sk-" + name, Name: name, GroupID: groupID, Status: service.StatusActive}
		require.NoError(t, repo.Create(ctx, key))
		return key.ID
	}
	alphaFirst := createKey("alpha-first", &alpha.ID)
	alphaSecond := createKey("alpha-second", &alpha.ID)
	zuluFirst := createKey("zulu-first", &zulu.ID)
	zuluSecond := createKey("zulu-second", &zulu.ID)
	ungroupedFirst := createKey("ungrouped-first", nil)
	ungroupedSecond := createKey("ungrouped-second", nil)

	for _, tc := range []struct {
		name  string
		order string
		want  []int64
	}{
		{name: "ascending", order: "asc", want: []int64{alphaFirst, alphaSecond, zuluFirst, zuluSecond, ungroupedFirst, ungroupedSecond}},
		{name: "descending", order: "desc", want: []int64{zuluSecond, zuluFirst, alphaSecond, alphaFirst, ungroupedSecond, ungroupedFirst}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keys, _, err := repo.ListByUserID(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 20, SortBy: "group", SortOrder: tc.order}, service.APIKeyListFilters{})
			require.NoError(t, err)
			got := make([]int64, 0, len(keys))
			for _, key := range keys {
				got = append(got, key.ID)
				if key.GroupID != nil {
					require.NotNil(t, key.Group)
				}
			}
			require.Equal(t, tc.want, got)
		})
	}
}
