package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"anlapi/internal/pkg/pagination"
	middleware2 "anlapi/internal/server/middleware"
	"anlapi/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type redeemHistoryRepo struct {
	service.RedeemCodeRepository
	userID      int64
	params      pagination.PaginationParams
	legacyLimit int
}

func (r *redeemHistoryRepo) ListByUser(_ context.Context, userID int64, limit int) ([]service.RedeemCode, error) {
	r.userID, r.legacyLimit = userID, limit
	return []service.RedeemCode{}, nil
}

func (r *redeemHistoryRepo) ListByUserPaginated(_ context.Context, userID int64, params pagination.PaginationParams, _ string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	r.userID, r.params = userID, params
	return []service.RedeemCode{}, &pagination.PaginationResult{Total: 101}, nil
}

func TestRedeemHistoryPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, query        string
		userID             int64
		status, page, size int
	}{
		{name: "legacy", query: "", userID: 7, status: 200},
		{name: "default", query: "?page=1", userID: 7, status: 200, page: 1, size: 20},
		{name: "size only", query: "?page_size=50", userID: 7, status: 200, page: 1, size: 50},
		{name: "cap", query: "?page_size=101", userID: 7, status: 200, page: 1, size: 100},
		{name: "overflow", query: "?page=9223372036854775807", userID: 7, status: 400},
		{name: "zero", query: "?page=0", userID: 7, status: 400},
		{name: "invalid", query: "?page=abc", userID: 7, status: 400},
		{name: "unauthenticated", query: "?page=1", status: 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &redeemHistoryRepo{}
			h := NewRedeemHandler(service.NewRedeemService(repo, nil, nil, nil, nil, nil, nil))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/api/v1/redeem/history"+tt.query, nil)
			if tt.userID != 0 {
				c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: tt.userID})
			}

			h.GetHistory(c)
			require.Equal(t, tt.status, w.Code)
			if tt.status != 200 {
				require.Zero(t, repo.userID)
				return
			}
			require.Equal(t, tt.userID, repo.userID)
			var body struct {
				Data json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			if tt.query == "" {
				require.Equal(t, 25, repo.legacyLimit)
				require.JSONEq(t, "[]", string(body.Data))
				return
			}
			require.Equal(t, tt.page, repo.params.Page)
			require.Equal(t, tt.size, repo.params.PageSize)
			var data struct {
				Items []service.RedeemCode `json:"items"`
				Total int64                `json:"total"`
				Page  int                  `json:"page"`
				Size  int                  `json:"page_size"`
			}
			require.NoError(t, json.Unmarshal(body.Data, &data))
			require.NotNil(t, data.Items)
			require.Equal(t, int64(101), data.Total)
			require.Equal(t, tt.page, data.Page)
			require.Equal(t, tt.size, data.Size)
		})
	}
}
