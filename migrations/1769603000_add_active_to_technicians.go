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

		if collection.Fields.GetByName("active") == nil {
			collection.Fields.Add(&core.BoolField{
				Name:     "active",
				Required: false,
			})
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Set all existing technicians to active=true
		records, err := app.FindRecordsByFilter("technicians", "active=false", "", 0, 0, nil)
		if err != nil {
			return nil // Ignore if fetch fails, maybe empty
		}

		for _, record := range records {
			record.Set("active", true)
			app.Save(record)
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}
