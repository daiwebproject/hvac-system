package handlers

import (
	"github.com/pocketbase/pocketbase/core"
)

// ShowFCMTest renders the FCM test/debug page
func (h *AdminHandler) ShowFCMTest(e *core.RequestEvent) error {
	// Hardcoded config for testing
	data := map[string]interface{}{
		"FirebaseConfig": map[string]string{
			"apiKey":            "AIzaSyB1zmMjyK6XtqVm8Kcu-EwwAUpfSTkg8AA",
			"projectId":         "techapp-hvac",
			"messagingSenderId": "250596752999",
			"appId":             "1:250596752999:web:6d810cf577eedfb7d55ec2",
		},
		"VapidPublicKey": "BM0Uvapd87utXwp2bBC_23HMT3LjtSwWGq6rUU8FnK6DvnJnTDCR_Kj4mGAC-HLgoia-tgjobgSWDpDJkKX_DBk",
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/fcm_test.html", data)
}
