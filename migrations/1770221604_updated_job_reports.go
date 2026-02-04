package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2600547242")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"hidden": false,
			"id": "file4078979840",
			"maxSelect": 10,
			"maxSize": 10,
			"mimeTypes": [
				"image/png",
				"image/jpeg",
				"image/webp"
			],
			"name": "after_images",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [
				"480x720"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"hidden": false,
			"id": "file2344219154",
			"maxSelect": 10,
			"maxSize": 10,
			"mimeTypes": [
				"image/png",
				"image/jpeg",
				"image/webp"
			],
			"name": "before_images",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [
				"480x720"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2600547242")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"hidden": false,
			"id": "file4078979840",
			"maxSelect": 10,
			"maxSize": 0,
			"mimeTypes": [
				"image/png",
				"image/jpeg",
				"image/webp"
			],
			"name": "after_images",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [
				"480x720"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"hidden": false,
			"id": "file2344219154",
			"maxSelect": 10,
			"maxSize": 0,
			"mimeTypes": [
				"image/png",
				"image/jpeg",
				"image/webp"
			],
			"name": "before_images",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": [
				"480x720"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
