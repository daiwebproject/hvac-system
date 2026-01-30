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

		if bookings.Fields.GetByName("booking_time") == nil {
			bookings.Fields.Add(&core.DateField{
				Name:     "booking_time",
				Required: false,
			})
		}

		return app.Save(bookings)
	}, func(app core.App) error {
		return nil
	})
}
