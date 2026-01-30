package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Create inventory_items collection
		collection := core.NewBaseCollection("inventory_items")

		// Item name (e.g., "Tụ điện 50uF")
		collection.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})

		// SKU/Part number for inventory tracking
		collection.Fields.Add(&core.TextField{
			Name: "sku",
		})

		// Category (Capacitors, Gas, Copper Pipes, etc.)
		collection.Fields.Add(&core.SelectField{
			Name:   "category",
			Values: []string{"capacitors", "gas", "pipes", "motors", "filters", "sensors", "other"},
		})

		// Unit price (VND)
		min := float64(0)
		collection.Fields.Add(&core.NumberField{
			Name:     "price",
			Required: true,
			Min:      &min,
		})

		// Stock quantity
		collection.Fields.Add(&core.NumberField{
			Name: "stock_quantity",
			Min:  &min,
		})

		// Unit of measurement (e.g., "cái", "kg", "mét")
		collection.Fields.Add(&core.TextField{
			Name: "unit",
		})

		// Description/Notes
		collection.Fields.Add(&core.TextField{
			Name: "description",
		})

		// Active flag
		collection.Fields.Add(&core.BoolField{
			Name: "is_active",
		})

		// Image (optional product photo)
		collection.Fields.Add(&core.FileField{
			Name:      "image",
			MaxSelect: 1,
			MaxSize:   5242880, // 5MB
		})

		// Index for fast SKU lookup
		collection.Indexes = []string{
			"CREATE INDEX idx_sku ON inventory_items (sku)",
			"CREATE INDEX idx_category ON inventory_items (category, is_active)",
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		// Create job_parts junction table
		jobParts := core.NewBaseCollection("job_parts")

		// Link to job report
		jobReports, err := app.FindCollectionByNameOrId("job_reports")
		if err == nil {
			jobParts.Fields.Add(&core.RelationField{
				Name:         "job_report_id",
				CollectionId: jobReports.Id,
				Required:     true,
			})
		}

		// Link to inventory item
		jobParts.Fields.Add(&core.RelationField{
			Name:         "item_id",
			CollectionId: collection.Id,
			Required:     true,
		})

		// Quantity used
		jobParts.Fields.Add(&core.NumberField{
			Name:     "quantity",
			Required: true,
			Min:      &min,
		})

		// Price at time of use (frozen price)
		jobParts.Fields.Add(&core.NumberField{
			Name:     "price_per_unit",
			Required: true,
			Min:      &min,
		})

		// Total = quantity * price_per_unit (calculated)
		jobParts.Fields.Add(&core.NumberField{
			Name: "total",
			Min:  &min,
		})

		return app.Save(jobParts)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("job_parts"); err == nil {
			app.Delete(collection)
		}
		if collection, err := app.FindCollectionByNameOrId("inventory_items"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
