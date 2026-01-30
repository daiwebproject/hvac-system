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

		field := collection.Fields.GetByName("job_status")
		if field != nil {
			selectField, ok := field.(*core.SelectField)
			if ok {
				// Append new options if not already present
				selectField.Values = []string{"pending", "assigned", "moving", "working", "in_progress", "completed", "cancelled"}
				// We overwrite to ensure order and completeness, assuming basic set
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
