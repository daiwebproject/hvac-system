package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Open Invoice Access (for Public View)
		invoices, err := app.FindCollectionByNameOrId("invoices")
		if err == nil {
			// Allow view if public_hash is present
			invoices.ViewRule = types.Pointer("public_hash != ''")
			if err := app.Save(invoices); err != nil {
				return err
			}
		}

		// 2. Open Job Reports Access (for Evidence Photos)
		reports, err := app.FindCollectionByNameOrId("job_reports")
		if err == nil {
			// Allow public view for now to ensure images load on invoice
			reports.ViewRule = types.Pointer("id != ''")
			if err := app.Save(reports); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}
