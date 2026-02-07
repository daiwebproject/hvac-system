package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Get required collections for relations
		techs, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}
		items, err := app.FindCollectionByNameOrId("inventory_items")
		if err != nil {
			return err
		}

		// ============================================
		// 1. Create TECH_INVENTORY collection
		// Tracks what each technician has in their truck
		// ============================================
		techInv, err := app.FindCollectionByNameOrId("tech_inventory")
		if err != nil {
			techInv = core.NewBaseCollection("tech_inventory")
			techInv.ListRule = types.Pointer("")
			techInv.ViewRule = types.Pointer("")
			techInv.CreateRule = types.Pointer("")
			techInv.UpdateRule = types.Pointer("")

			// Relations
			techInv.Fields.Add(&core.RelationField{
				Name:         "technician_id",
				CollectionId: techs.Id,
				Required:     true,
				MaxSelect:    1,
			})
			techInv.Fields.Add(&core.RelationField{
				Name:         "item_id",
				CollectionId: items.Id,
				Required:     true,
				MaxSelect:    1,
			})

			// Data
			techInv.Fields.Add(&core.NumberField{
				Name:     "quantity",
				Required: true,
				Min:      types.Pointer(0.0),
			})

			// Unique constraint: one record per tech+item combo
			techInv.AddIndex("idx_tech_item_unique", true, "technician_id, item_id", "")

			if err := app.Save(techInv); err != nil {
				return err
			}
		}

		// ============================================
		// 2. Create STOCK_TRANSFERS collection
		// Audit log for all stock movements
		// ============================================
		transfers, err := app.FindCollectionByNameOrId("stock_transfers")
		if err != nil {
			transfers = core.NewBaseCollection("stock_transfers")
			transfers.ListRule = types.Pointer("")
			transfers.ViewRule = types.Pointer("")
			transfers.CreateRule = types.Pointer("")

			// Transfer type: main_to_tech, tech_to_main, tech_to_job
			transfers.Fields.Add(&core.TextField{
				Name:     "transfer_type",
				Required: true,
			})

			// Source and destination IDs (tech ID or null for main stock)
			transfers.Fields.Add(&core.TextField{Name: "from_id"})
			transfers.Fields.Add(&core.TextField{Name: "to_id"})

			// Item and quantity
			transfers.Fields.Add(&core.RelationField{
				Name:         "item_id",
				CollectionId: items.Id,
				Required:     true,
				MaxSelect:    1,
			})
			transfers.Fields.Add(&core.NumberField{
				Name:     "quantity",
				Required: true,
			})

			// Metadata
			transfers.Fields.Add(&core.TextField{Name: "note"})
			transfers.Fields.Add(&core.TextField{Name: "created_by"}) // Admin who made transfer
			transfers.Fields.Add(&core.TextField{Name: "job_id"})     // If transfer is for a job

			// Indexes
			transfers.AddIndex("idx_transfers_item", false, "item_id", "")
			transfers.AddIndex("idx_transfers_tech", false, "to_id", "")

			if err := app.Save(transfers); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}
