package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// UpdateAppGroupRequest selects the desktop App group; 0 follows the system default.
type UpdateAppGroupRequest struct {
	GroupID int64 `json:"group_id"`
}

// GetMyAppGroup returns the group the current user picked for the desktop App.
// GET /api/v1/user/app-group
func (h *APIKeyHandler) GetMyAppGroup(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	groupID, err := h.apiKeyService.GetUserAppGroupID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID})
}

// UpdateMyAppGroup sets the group the current user's desktop App uses.
// PUT /api/v1/user/app-group
func (h *APIKeyHandler) UpdateMyAppGroup(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req UpdateAppGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.apiKeyService.SetUserAppGroupID(c.Request.Context(), subject.UserID, req.GroupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": req.GroupID})
}
