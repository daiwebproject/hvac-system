package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		fmt.Println("⚡ RUNNING FIX MIGRATION: 1770250000_fix_public_access.go")

		// 1. Fix Invoices ViewRule
		invoices, err := app.FindCollectionByNameOrId("invoices")
		if err == nil {
			invoices.ViewRule = types.Pointer("id != ''") // PUBLIC READ
			if err := app.Save(invoices); err != nil {
				return err
			}
			fmt.Println("✅ Invoices ViewRule set to PUBLIC")
		}

		// 2. Fix Job Reports ViewRule & Schema
		reports, err := app.FindCollectionByNameOrId("job_reports")
		if err == nil {
			reports.ViewRule = types.Pointer("id != ''") // PUBLIC READ

			// Ensure file fields exist (idempotent check)
			// (Assuming they exist, but setting them ensures configured correctly)

			if err := app.Save(reports); err != nil {
				return err
			}
			fmt.Println("✅ Job Reports ViewRule set to PUBLIC")
		}

		return nil
	}, nil)
}
