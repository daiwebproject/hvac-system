package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("invoice_items")
		if err == nil {
			return nil // already exists
		}

		invoicesColl, err := app.FindCollectionByNameOrId("invoices")
		if err != nil {
			return err
		}

		collection = core.NewBaseCollection("invoice_items")

		collection.ListRule = types.Pointer("")
		collection.ViewRule = types.Pointer("")
		collection.CreateRule = types.Pointer("")
		collection.UpdateRule = types.Pointer("")

		collection.Fields.Add(&core.RelationField{
			Name:          "invoice_id",
			MaxSelect:     1,
			CollectionId:  invoicesColl.Id,
			CascadeDelete: true,
		})
		collection.Fields.Add(&core.TextField{
			Name: "item_name",
		})
		collection.Fields.Add(&core.NumberField{
			Name: "quantity",
		})
		collection.Fields.Add(&core.NumberField{
			Name: "unit_price",
		})
		collection.Fields.Add(&core.NumberField{
			Name: "total",
		})

		return app.Save(collection)
	}, nil)
}
