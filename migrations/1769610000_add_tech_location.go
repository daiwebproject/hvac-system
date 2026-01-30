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

		// Add Current Latitude
		if collection.Fields.GetByName("current_lat") == nil {
			collection.Fields.Add(&core.NumberField{
				Name: "current_lat",
			})
		}

		// Add Current Longitude
		if collection.Fields.GetByName("current_long") == nil {
			collection.Fields.Add(&core.NumberField{
				Name: "current_long",
			})
		}

		// Add Last Updated Timestamp
		if collection.Fields.GetByName("location_updated_at") == nil {
			collection.Fields.Add(&core.DateField{
				Name: "location_updated_at",
			})
		}

		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
