package middleware

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// RequireAdmin middleware ensures the user is an authenticated Admin
func RequireAdmin(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		cookie, err := e.Request.Cookie("pb_auth")
		if err != nil || cookie.Value == "" {
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		record, err := app.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
		if err != nil || record == nil || !record.IsSuperuser() {
			return e.Redirect(http.StatusSeeOther, "/login")
		}
		e.Auth = record
		return e.Next()
	}
}

// RequireTech middleware ensures the user is an authenticated Technician
func RequireTech(app *pocketbase.PocketBase) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		cookie, err := e.Request.Cookie("pb_auth")
		if err != nil || cookie.Value == "" {
			return e.Redirect(http.StatusSeeOther, "/tech/login")
		}

		// Verify token against 'technicians' collection
		record, err := app.FindAuthRecordByToken(cookie.Value, core.TokenTypeAuth)
		if err != nil || record == nil || record.Collection().Name != "technicians" {
			return e.Redirect(http.StatusSeeOther, "/tech/login")
		}

		e.Auth = record
		return e.Next()
	}
}
