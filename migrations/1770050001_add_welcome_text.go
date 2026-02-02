package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Add welcome_text field if it doesn't exist
		if collection.Fields.GetByName("welcome_text") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "welcome_text",
				Max:  1000,
			})
			return app.Save(collection)
		}

		return nil
	}, nil)
}
