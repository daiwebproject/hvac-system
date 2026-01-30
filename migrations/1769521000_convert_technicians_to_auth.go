package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Remove old relation from bookings
		bookings, err := app.FindCollectionByNameOrId("bookings")
		// Only proceed if collection exists (err == nil)
		if err == nil {
			if f := bookings.Fields.GetByName("technician_id"); f != nil {
				bookings.Fields.RemoveByName("technician_id")
				if err := app.Save(bookings); err != nil {
					return err
				}
			}
		}

		// 1.5 Remove relation from time_slots if exists
		timeSlots, err := app.FindCollectionByNameOrId("time_slots")
		if err == nil {
			if f := timeSlots.Fields.GetByName("technician_id"); f != nil {
				timeSlots.Fields.RemoveByName("technician_id")
				app.Save(timeSlots)
			}
		}

		// 2. Delete old technicians collection (base)
		oldTechs, err := app.FindCollectionByNameOrId("technicians")
		if err == nil {
			if err := app.Delete(oldTechs); err != nil {
				return err
			}
		}

		// 3. Create new technicians collection (auth)
		// Check if it already exists (if partial migration happened)
		if _, err := app.FindCollectionByNameOrId("technicians"); err != nil {
			newTechs := core.NewAuthCollection("technicians")
			newTechs.Fields.Add(&core.TextField{Name: "name", Required: true})
			newTechs.Fields.Add(&core.TextField{Name: "phone"})
			newTechs.Fields.Add(&core.TextField{Name: "skills"}) // e.g. "AC, Washer"

			if err := app.Save(newTechs); err != nil {
				return err
			}
		}

		// 4. Re-add relation to bookings
		// Need to get the new 'technicians' id
		newTechs, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}

		bookings, err = app.FindCollectionByNameOrId("bookings")
		if err != nil {
			return err
		}

		// Check if relation already exists
		if bookings.Fields.GetByName("technician_id") == nil {
			bookings.Fields.Add(&core.RelationField{
				Name:         "technician_id",
				CollectionId: newTechs.Id,
				MaxSelect:    1,
			})
			return app.Save(bookings)
		}

		return nil

	}, func(app core.App) error {
		return nil
	})
}
