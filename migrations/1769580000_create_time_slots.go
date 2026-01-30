package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create time_slots collection
		collection := core.NewBaseCollection("time_slots")

		// Date (e.g., "2026-01-28")
		collection.Fields.Add(&core.TextField{
			Name:     "date",
			Required: true,
		})

		// Start time (e.g., "08:00")
		collection.Fields.Add(&core.TextField{
			Name:     "start_time",
			Required: true,
		})

		// End time (e.g., "10:00")
		collection.Fields.Add(&core.TextField{
			Name:     "end_time",
			Required: true,
		})

		// Technician assigned to this slot (optional, for future tech-specific scheduling)
		technicians, err := app.FindCollectionByNameOrId("technicians")
		if err == nil {
			collection.Fields.Add(&core.RelationField{
				Name:         "technician_id",
				CollectionId: technicians.Id,
				MaxSelect:    1,
			})
		}

		// Is this slot booked?
		collection.Fields.Add(&core.BoolField{
			Name: "is_booked",
		})

		// Booking reference (if booked)
		bookings, err := app.FindCollectionByNameOrId("bookings")
		if err == nil {
			collection.Fields.Add(&core.RelationField{
				Name:         "booking_id",
				CollectionId: bookings.Id,
				MaxSelect:    1,
			})
		}

		// Max capacity per slot (number of techs available in this time window)
		min := float64(1)
		collection.Fields.Add(&core.NumberField{
			Name: "max_capacity",
			Min:  &min,
		})

		// Current bookings count
		collection.Fields.Add(&core.NumberField{
			Name: "current_bookings",
		})

		// Create index for efficient querying of available slots
		collection.Indexes = []string{
			"CREATE INDEX idx_date_time ON time_slots (date, start_time)",
			"CREATE INDEX idx_availability ON time_slots (is_booked, date)",
		}

		return app.Save(collection)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("time_slots"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
