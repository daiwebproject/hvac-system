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
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"hidden": false,
			"id": "file2344219154",
			"maxSelect": 99,
			"maxSize": 0,
			"mimeTypes": [
				"image/png",
				"image/jpeg",
				"video/mp4",
				"video/quicktime",
				"image/vnd.mozilla.apng"
			],
			"name": "before_images",
			"presentable": false,
			"protected": true,
			"required": true,
			"system": false,
			"thumbs": [
				"50x50"
			],
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"hidden": false,
			"id": "file4078979840",
			"maxSelect": 99,
			"maxSize": 0,
			"mimeTypes": [
				"image/png",
				"image/jpeg",
				"video/mp4",
				"video/quicktime",
				"image/vnd.mozilla.apng"
			],
			"name": "after_images",
			"presentable": false,
			"protected": true,
			"required": true,
			"system": false,
			"thumbs": [
				"50x50"
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
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "file2344219154",
			"maxSelect": 10,
			"maxSize": 10485760,
			"mimeTypes": null,
			"name": "before_images",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": null,
			"type": "file"
		}`)); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"hidden": false,
			"id": "file4078979840",
			"maxSelect": 10,
			"maxSize": 10485760,
			"mimeTypes": null,
			"name": "after_images",
			"presentable": false,
			"protected": false,
			"required": true,
			"system": false,
			"thumbs": null,
			"type": "file"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
