package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// ============================================
		// 1. Update INVENTORY_ITEMS (Products) with new fields
		// ============================================
		items, err := app.FindCollectionByNameOrId("inventory_items")
		if err != nil {
			return err
		}

		// Add min_threshold field if not exists
		if items.Fields.GetByName("min_threshold") == nil {
			items.Fields.Add(&core.NumberField{
				Name: "min_threshold",
				Min:  types.Pointer(0.0),
			})
		}

		// Add price_import (giá vốn) if not exists
		if items.Fields.GetByName("price_import") == nil {
			items.Fields.Add(&core.NumberField{
				Name: "price_import",
				Min:  types.Pointer(0.0),
			})
		}

		// Rename existing 'price' to 'price_sell' conceptually
		// (keeping 'price' for backward compatibility, price_sell = price)

		if err := app.Save(items); err != nil {
			return err
		}

		// ============================================
		// 2. Create INVENTORY_STOCKS (Main + Tech stocks)
		// ============================================
		techs, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}

		stocks, err := app.FindCollectionByNameOrId("inventory_stocks")
		if err != nil {
			stocks = core.NewBaseCollection("inventory_stocks")
			stocks.ListRule = types.Pointer("")
			stocks.ViewRule = types.Pointer("")
			stocks.CreateRule = types.Pointer("")
			stocks.UpdateRule = types.Pointer("")

			// Relations
			stocks.Fields.Add(&core.RelationField{
				Name:         "product_id",
				CollectionId: items.Id,
				Required:     true,
				MaxSelect:    1,
			})
			stocks.Fields.Add(&core.RelationField{
				Name:         "tech_id",
				CollectionId: techs.Id,
				Required:     false, // NULL = main warehouse
				MaxSelect:    1,
			})

			// Quantity
			stocks.Fields.Add(&core.NumberField{
				Name:     "quantity",
				Required: true,
				Min:      types.Pointer(0.0),
			})

			// Unique constraint: one record per product+location combo
			stocks.AddIndex("idx_stock_product_tech", true, "product_id, tech_id", "")

			if err := app.Save(stocks); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}
