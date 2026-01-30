package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Cleanup: Delete if exists to ensure schema consistency
		existing, err := app.FindCollectionByNameOrId("invoices")
		if err == nil && existing != nil {
			if err := app.Delete(existing); err != nil {
				return err
			}
		}

		// Create invoices collection
		collection := core.NewBaseCollection("invoices")

		// Link to booking
		bookings, err := app.FindCollectionByNameOrId("bookings")
		if err == nil {
			collection.Fields.Add(&core.RelationField{
				Name:         "booking_id",
				CollectionId: bookings.Id,
				Required:     true,
				MaxSelect:    1,
			})
		}

		// Financial fields
		min := float64(0)
		collection.Fields.Add(&core.NumberField{
			Name: "parts_total",
			Min:  &min,
		})
		collection.Fields.Add(&core.NumberField{
			Name: "labor_total",
			Min:  &min,
		})
		collection.Fields.Add(&core.NumberField{
			Name: "discount",
			Min:  &min,
		})
		collection.Fields.Add(&core.NumberField{
			Name:     "total_amount",
			Required: true,
			Min:      &min,
		})

		// Status
		collection.Fields.Add(&core.SelectField{
			Name:     "status",
			Values:   []string{"unpaid", "paid", "cancelled"},
			Required: true,
		})

		// Payment Method
		collection.Fields.Add(&core.SelectField{
			Name:   "payment_method",
			Values: []string{"cash", "transfer", "card"},
		})

		// Notes
		collection.Fields.Add(&core.TextField{
			Name: "notes",
		})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("invoices")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
