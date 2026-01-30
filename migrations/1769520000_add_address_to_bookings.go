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

		// Add address_details if missing
		if bookings.Fields.GetByName("address_details") == nil {
			bookings.Fields.Add(&core.TextField{Name: "address_details"})
		}

		return app.Save(bookings)
	}, func(app core.App) error {
		return nil
	})
}
