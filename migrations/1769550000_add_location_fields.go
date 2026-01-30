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

		// Add Latitude
		if bookings.Fields.GetByName("lat") == nil {
			bookings.Fields.Add(&core.NumberField{
				Name: "lat",
			})
		}

		// Add Longitude
		if bookings.Fields.GetByName("long") == nil {
			bookings.Fields.Add(&core.NumberField{
				Name: "long",
			})
		}

		return app.Save(bookings)
	}, func(app core.App) error {
		return nil
	})
}
