package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_986407980")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_2466930996",
			"hidden": false,
			"id": "relation3871724694",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "technician_id",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_986407980")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation3871724694")

		return app.Save(collection)
	})
}
