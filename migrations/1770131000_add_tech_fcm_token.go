package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}

		// Check if field already exists (idempotency)
		if collection.Fields.GetByName("fcm_token") != nil {
			return nil
		}

		collection.Fields.Add(&core.TextField{
			Name: "fcm_token",
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}

		if field := collection.Fields.GetByName("fcm_token"); field != nil {
			collection.Fields.RemoveByName("fcm_token")
			return app.Save(collection)
		}

		return nil
	})
}
