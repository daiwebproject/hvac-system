package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. DELETE E-COMMERCE COLLECTIONS (Order matters due to relations)
		// Delete children first: order_items, reviews, time_slots
		// Then parents: products, orders
		collectionsToDelete := []string{"order_items", "reviews", "time_slots", "products", "orders", "carts"}
		for _, name := range collectionsToDelete {
			if col, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(col); err != nil {
					// We might want to log error but for now return
					return err
				}
			}
		}

		// 2. ENHANCE SERVICES COLLECTION
		services, err := app.FindCollectionByNameOrId("services")
		if err != nil {
			return err
		}

		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		// Check and Add 'category_id'
		if services.Fields.GetByName("category_id") == nil {
			services.Fields.Add(&core.RelationField{
				Name:         "category_id",
				CollectionId: categories.Id,
				MaxSelect:    1,
			})
		}

		// Check and Add 'image' field for Service visual
		if services.Fields.GetByName("image") == nil {
			services.Fields.Add(&core.FileField{
				Name:      "image",
				MaxSelect: 1,
				MaxSize:   5242880,
			})
		}

		// Check and Add 'duration_minutes' for approximate time
		if services.Fields.GetByName("duration_minutes") == nil {
			min := float64(0)
			services.Fields.Add(&core.NumberField{
				Name: "duration_minutes",
				Min:  &min,
			})
		}

		return app.Save(services)

	}, func(app core.App) error {
		// No rollback logic for destructive cleanup in this context
		// as we want to permanently switch mode.
		return nil
	})
}
