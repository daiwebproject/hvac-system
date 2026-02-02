package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"hvac-system/internal/adapter/repository"

	"github.com/pocketbase/pocketbase/core"
)

// SettingsMiddleware loads global settings into the request context
// and enforces license expiration logic.
func SettingsMiddleware(settingsRepo *repository.SettingsRepo) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// 1. Fetch Settings
		settings, err := settingsRepo.GetSettings()
		if err != nil {
			// Log error but generally continue with defaults (Repo handles defaults)
			fmt.Printf("Middleware Warning: Failed to load settings: %v\n", err)
		}

		// 2. Store in Context for RenderPage
		e.Set("Settings", settings)

		// [OPTIMIZATION] Pre-calculate Logo URL with thumbnail to reduce load
		if settings.Logo != "" {
			// Use 'settings' collection name as ID (PocketBase supports name or ID)
			logoUrl := fmt.Sprintf("/api/files/settings/%s/%s?thumb=200x0", settings.Id, settings.Logo)
			e.Set("LogoUrl", logoUrl)
		}

		// 3. License Gatekeeper Logic
		// Skip check for static assets, login pages, and admin pages (to allow fixing license)
		path := e.Request.URL.Path
		isStatic := strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/_/")
		isAuth := strings.HasPrefix(path, "/login") || strings.HasPrefix(path, "/admin/login") || strings.HasPrefix(path, "/tech/login")
		isAdmin := strings.HasPrefix(path, "/admin") // Allow admin to access dashboard to update license

		if !isStatic && !isAuth && !isAdmin {
			if settings.ExpiryDate != "" {
				expiry, err := time.Parse("2006-01-02", settings.ExpiryDate)
				if err == nil && time.Now().After(expiry) {
					// License Expired
					return e.String(http.StatusPaymentRequired, "⚠ LICENSE EXPIRED. Please contact administrator to renew your subscription.")
					// Alternatively, render a nice "payment_required.html" page
				}
			}
		}

		return e.Next()
	}
}
