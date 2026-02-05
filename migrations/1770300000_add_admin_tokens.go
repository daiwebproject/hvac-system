package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Add admin_fcm_tokens column (JSON) to store list of tokens
		collection.Fields.Add(&core.JSONField{Name: "admin_fcm_tokens"})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("admin_fcm_tokens")

		return app.Save(collection)
	})
}
