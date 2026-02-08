package migrations

import (
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Check if any admin exists
		totalAdmins, err := app.CountRecords("_superusers")
		if err != nil {
			return err
		}

		if totalAdmins > 0 {
			return nil // Already initialized
		}

		// 2. Get credentials from ENV
		email := os.Getenv("INITIAL_ADMIN_EMAIL")
		pass := os.Getenv("INITIAL_ADMIN_PASSWORD")

		if email == "" || pass == "" {
			return nil // No env vars set, skip auto-creation (let user do it via UI)
		}

		// 3. Create Admin
		collection, err := app.FindCollectionByNameOrId("_superusers")
		if err != nil {
			return err
		}

		record := core.NewRecord(collection)
		record.Set("email", email)
		record.Set("password", pass)

		return app.Save(record)
	}, nil)
}
