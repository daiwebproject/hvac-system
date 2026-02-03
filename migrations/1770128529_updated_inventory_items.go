package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1406054367")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_711030668",
			"hidden": false,
			"id": "relation696906237",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "invoice_id",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text2517842685",
			"max": 0,
			"min": 0,
			"name": "item_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
			"hidden": false,
			"id": "number2683508278",
			"max": null,
			"min": null,
			"name": "quantity",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"hidden": false,
			"id": "number1106926802",
			"max": null,
			"min": null,
			"name": "unit_price",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"hidden": false,
			"id": "number3257917790",
			"max": null,
			"min": null,
			"name": "total",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1406054367")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation696906237")

		// remove field
		collection.Fields.RemoveById("text2517842685")

		// remove field
		collection.Fields.RemoveById("number2683508278")

		// remove field
		collection.Fields.RemoveById("number1106926802")

		// remove field
		collection.Fields.RemoveById("number3257917790")

		return app.Save(collection)
	})
}
