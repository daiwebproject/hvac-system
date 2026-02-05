package handlers

import (
	"github.com/pocketbase/pocketbase/core"
)

// ShowFCMTest renders the FCM test/debug page
func (h *AdminHandler) ShowFCMTest(e *core.RequestEvent) error {
	settings, err := h.SettingsRepo.GetSettings()
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"FirebaseConfig": settings.FirebaseConfig,
		"VapidPublicKey": settings.VapidPublicKey,
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/fcm_test.html", data)
}
