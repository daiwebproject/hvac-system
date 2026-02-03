package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		invoices, err := app.FindCollectionByNameOrId("invoices")
		if err != nil {
			return err
		}

		// Find the "total_amount" field and update it
		field := invoices.Fields.GetByName("total_amount")
		if field != nil {
			// Type assertion to NumberField to modify properties
			if numField, ok := field.(*core.NumberField); ok {
				numField.Required = false
			}
		}

		return app.Save(invoices)
	}, nil)
}
