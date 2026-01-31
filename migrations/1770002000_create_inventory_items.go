package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Create INVENTORY_ITEMS collection (Correcting 'inventory' mismatch)
		items, err := app.FindCollectionByNameOrId("inventory_items")
		if err != nil {
			items = core.NewBaseCollection("inventory_items")
			items.ListRule = types.Pointer("")
			items.ViewRule = types.Pointer("")
			items.CreateRule = types.Pointer("")
			items.UpdateRule = types.Pointer("")

			items.Fields.Add(&core.TextField{Name: "name", Required: true})
			items.Fields.Add(&core.TextField{Name: "sku"})
			items.Fields.Add(&core.TextField{Name: "category"})
			items.Fields.Add(&core.NumberField{Name: "price"})
			items.Fields.Add(&core.NumberField{Name: "stock_quantity"})
			items.Fields.Add(&core.TextField{Name: "unit"})
			items.Fields.Add(&core.TextField{Name: "description"})
			items.Fields.Add(&core.BoolField{Name: "is_active"})

			// Indexes
			items.AddIndex("idx_items_sku", true, "sku", "")
			items.AddIndex("idx_items_category", false, "category", "")

			if err := app.Save(items); err != nil {
				return err
			}
		}

		// 2. Create JOB_PARTS collection (Missing in initial schema)
		parts, err := app.FindCollectionByNameOrId("job_parts")
		if err != nil {
			// Fetch dependencies
			jobReports, err := app.FindCollectionByNameOrId("job_reports")
			if err != nil {
				return err
			}

			parts = core.NewBaseCollection("job_parts")
			// Rules can be restricted later, open for now for dev
			parts.ListRule = types.Pointer("")
			parts.ViewRule = types.Pointer("")
			parts.CreateRule = types.Pointer("")

			// Relations
			parts.Fields.Add(&core.RelationField{
				Name:         "job_report_id",
				CollectionId: jobReports.Id, // Must use valid Collection ID
				Required:     true,
				MaxSelect:    1,
			})
			parts.Fields.Add(&core.RelationField{
				Name:         "item_id",
				CollectionId: items.Id,
				Required:     true,
				MaxSelect:    1,
			})

			// Data
			parts.Fields.Add(&core.NumberField{Name: "quantity"})
			parts.Fields.Add(&core.NumberField{Name: "price_per_unit"})
			parts.Fields.Add(&core.NumberField{Name: "total"})

			// Index
			parts.AddIndex("idx_parts_report", false, "job_report_id", "")

			if err := app.Save(parts); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}
