package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Update BOOKINGS collection
		bookings, err := app.FindCollectionByNameOrId("bookings")
		if err != nil {
			return err
		}

		// Add new fields
		bookings.Fields.Add(&core.DateField{Name: "arrived_at"})
		bookings.Fields.Add(&core.DateField{Name: "started_at"})
		bookings.Fields.Add(&core.DateField{Name: "completed_at"})
		bookings.Fields.Add(&core.TextField{Name: "cancel_reason"})
		bookings.Fields.Add(&core.FileField{
			Name:      "customer_signature",
			MaxSelect: 1,
		})

		if err := app.Save(bookings); err != nil {
			return err
		}

		// 2. Update INVOICES collection
		invoices, err := app.FindCollectionByNameOrId("invoices")
		if err != nil {
			return err
		}

		invoices.Fields.Add(&core.TextField{Name: "public_hash"})
		// Add index for fast lookup by hash
		invoices.AddIndex("idx_invoices_hash", true, "public_hash", "")

		if err := app.Save(invoices); err != nil {
			return err
		}

		return nil
	}, nil)
}
