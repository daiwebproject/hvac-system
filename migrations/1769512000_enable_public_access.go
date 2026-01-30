package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		collections := []string{"categories", "products"}

		for _, name := range collections {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}

			// Empty string means public access
			collection.ListRule = types.Pointer("")
			collection.ViewRule = types.Pointer("")

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}
