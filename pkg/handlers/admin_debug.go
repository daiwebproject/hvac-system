package handlers

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// DebugAdminTokens shows current admin FCM tokens (for debugging only)
// GET /admin/debug/fcm-tokens
func (h *AdminHandler) DebugAdminTokens(e *core.RequestEvent) error {
	settings, err := h.SettingsRepo.GetSettings()
	if err != nil {
		return e.JSON(500, map[string]interface{}{
			"error": "Failed to get settings",
		})
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"admin_fcm_tokens": settings.AdminFCMTokens,
		"token_count":      len(settings.AdminFCMTokens),
	})
}
