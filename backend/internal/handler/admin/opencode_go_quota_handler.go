package admin

import (
	"net/http"
	"strconv"

	"anlapi/internal/pkg/response"
	"anlapi/internal/service"
	"github.com/gin-gonic/gin"
)

// OpenCodeGoQuotaHandler exposes a read-only admin probe for OpenCode Go
// subscription windows. Credentials never leave the service layer.
type OpenCodeGoQuotaHandler struct {
	quotaService *service.OpenCodeGoQuotaService
}

func NewOpenCodeGoQuotaHandler(quotaService *service.OpenCodeGoQuotaService) *OpenCodeGoQuotaHandler {
	return &OpenCodeGoQuotaHandler{quotaService: quotaService}
}

// QueryQuota queries and persists the latest OpenCode Go usage snapshot.
// GET /api/v1/admin/opencode/accounts/:id/quota
func (h *OpenCodeGoQuotaHandler) QueryQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h == nil || h.quotaService == nil {
		response.Error(c, http.StatusServiceUnavailable, "OpenCode Go quota service is not enabled")
		return
	}
	result, err := h.quotaService.QueryUsage(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
