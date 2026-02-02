package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// ----------------------------------------------------
		// SETTINGS COLLECTION
		// ----------------------------------------------------
		settings := core.NewBaseCollection("settings")

		// System fields
		settings.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		settings.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})

		// Rules: Only admins can view/edit for now, or maybe public view if needed for frontend?
		// Let's allow public view for these settings since they are company info.
		settings.ListRule = types.Pointer("")
		settings.ViewRule = types.Pointer("")
		settings.CreateRule = nil // Only created via migration (or admin)
		settings.DeleteRule = nil // Should not be deleted

		// --- brand ---
		settings.Fields.Add(&core.TextField{Name: "company_name", Required: true})
		settings.Fields.Add(&core.FileField{Name: "logo", MaxSelect: 1})
		settings.Fields.Add(&core.TextField{Name: "hotline"})

		// --- finance ---
		settings.Fields.Add(&core.TextField{Name: "bank_bin"})     // e.g. 970422
		settings.Fields.Add(&core.TextField{Name: "bank_account"}) // e.g. 0333666999
		settings.Fields.Add(&core.TextField{Name: "bank_owner"})   // e.g. CTY DIEN LANH
		settings.Fields.Add(&core.TextField{Name: "qr_template"})  // compact/full

		// --- license ---
		settings.Fields.Add(&core.TextField{Name: "license_key"})
		settings.Fields.Add(&core.TextField{Name: "expiry_date"}) // Simple Text for now

		if err := app.Save(settings); err != nil {
			return err
		}

		// ----------------------------------------------------
		// SEED DEFAULT RECORD
		// ----------------------------------------------------
		// Only create if empty (which it is, since we just created the table)
		record := core.NewRecord(settings)
		record.Set("company_name", "HVAC System")
		record.Set("hotline", "0909000111")
		// Bank defaults
		record.Set("bank_bin", "970422")
		record.Set("bank_account", "0333666999")
		record.Set("bank_owner", "NGUYEN VAN A")
		record.Set("qr_template", "compact")
		// License defaults
		record.Set("license_key", "FREE-TRIAL")
		record.Set("expiry_date", "2099-12-31")

		return app.Save(record)

	}, nil)
}
