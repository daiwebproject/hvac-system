package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("bookings")
		if err != nil {
			return err
		}

		// Add moving_start_at field
		if collection.Fields.GetByName("moving_start_at") == nil {
			collection.Fields.Add(&core.DateField{
				Name:     "moving_start_at",
				Required: false,
			})
		}

		// Add working_start_at field
		if collection.Fields.GetByName("working_start_at") == nil {
			collection.Fields.Add(&core.DateField{
				Name:     "working_start_at",
				Required: false,
			})
		}

		// Add status field options if not updated
		// Ideally we should update the enum options for "job_status" if it's a select field
		// But for now, assuming it's a text/select field that accepts string values "moving", "working"

		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
