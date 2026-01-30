package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		bookings, err := app.FindCollectionByNameOrId("bookings")
		if err != nil {
			return err
		}

		// Add time_slot_id relation
		slots, err := app.FindCollectionByNameOrId("time_slots")
		if err == nil && bookings.Fields.GetByName("time_slot_id") == nil {
			bookings.Fields.Add(&core.RelationField{
				Name:         "time_slot_id",
				CollectionId: slots.Id,
				MaxSelect:    1,
			})
		}

		return app.Save(bookings)
	}, func(app core.App) error {
		return nil
	})
}
