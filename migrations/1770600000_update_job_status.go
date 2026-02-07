package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("bookings")
		if err != nil {
			return err
		}

		// Update job_status options
		field := collection.Fields.GetByName("job_status")
		if field == nil {
			return fmt.Errorf("field job_status not found")
		}

		selectField, ok := field.(*core.SelectField)
		if !ok {
			return fmt.Errorf("field job_status is not a SelectField")
		}

		// Check if "accepted" is already in the list
		hasAccepted := false
		for _, v := range selectField.Values {
			if v == "accepted" {
				hasAccepted = true
				break
			}
		}

		if !hasAccepted {
			fmt.Println("Migrating: Adding 'accepted' status to bookings collection")
			// Add "accepted" to the correct position (after "assigned")
			// Or just append, order in UI is usually handled by code/logic, but PB UI uses array order.
			// Rebuilding array to be neat:
			// pending, assigned, accepted, moving, arrived, working, completed, cancelled, failed
			newValues := []string{"pending", "assigned", "accepted", "moving", "arrived", "working", "completed", "cancelled", "failed"}
			selectField.Values = newValues

			return app.Save(collection)
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}
