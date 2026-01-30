package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Update Bookings Collection
		bookings, err := app.FindCollectionByNameOrId("bookings")
		if err != nil {
			return err
		}

		// Add new fields to bookings
		bookings.Fields.Add(&core.TextField{Name: "device_type"})
		bookings.Fields.Add(&core.TextField{Name: "brand"})
		bookings.Fields.Add(&core.TextField{Name: "issue_description"})
		bookings.Fields.Add(&core.FileField{
			Name:      "client_images",
			MaxSelect: 5,
			MaxSize:   5242880, // 5MB
		})

		// Make sure status has options we want (if it's a select, but currently it's text)
		// We can change it to Select or keep as Text. Usually Select is better for strict flow.
		// For now, let's update it to SelectField if possible, but converting type is hard.
		// Let's assume we keep it Text or add a new field 'job_status' if we want Select.
		// The plan said "job_status: Select". Let's add 'job_status' as Select and migrate data later if needed.
		// Actually, let's try to add 'job_status' as Select.
		bookings.Fields.Add(&core.SelectField{
			Name:      "job_status",
			Values:    []string{"pending", "assigned", "in_progress", "completed", "cancelled"},
			MaxSelect: 1,
		})

		bookings.Fields.Add(&core.NumberField{Name: "estimated_cost"})

		if err := app.Save(bookings); err != nil {
			return err
		}

		// 2. Create Job Reports Collection
		jobReports := core.NewBaseCollection("job_reports")
		jobReports.Fields.Add(&core.RelationField{
			Name:         "booking_id",
			CollectionId: bookings.Id,
			MaxSelect:    1,
			Required:     true,
		})
		jobReports.Fields.Add(&core.TextField{Name: "technician_notes"})
		jobReports.Fields.Add(&core.FileField{
			Name:      "proof_images",
			MaxSelect: 5,
			MaxSize:   5242880,
		})
		jobReports.Fields.Add(&core.JSONField{Name: "parts_replaced"})
		jobReports.Fields.Add(&core.NumberField{Name: "final_cost"})

		if err := app.Save(jobReports); err != nil {
			return err
		}

		// 3. Create Invoices Collection
		invoices := core.NewBaseCollection("invoices")
		invoices.Fields.Add(&core.RelationField{
			Name:         "booking_id",
			CollectionId: bookings.Id,
			MaxSelect:    1,
			Required:     true,
		})
		invoices.Fields.Add(&core.TextField{Name: "invoice_no"})
		invoices.Fields.Add(&core.NumberField{Name: "amount"})
		invoices.Fields.Add(&core.SelectField{
			Name:      "payment_method",
			Values:    []string{"cash", "transfer", "vietqr"},
			MaxSelect: 1,
		})
		invoices.Fields.Add(&core.SelectField{
			Name:      "status",
			Values:    []string{"unpaid", "paid"},
			MaxSelect: 1,
		})
		invoices.Fields.Add(&core.URLField{Name: "einvoice_url"})

		if err := app.Save(invoices); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Revert logic (optional for dev)
		return nil
	})
}
