package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. UPDATE CATEGORIES
		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}
		categories.Fields.Add(&core.RelationField{
			Name:         "parent_id",
			CollectionId: categories.Id, // Self-reference
			MaxSelect:    1,
		})
		categories.Fields.Add(&core.TextField{Name: "color"}) // Hex code
		if err := app.Save(categories); err != nil {
			return err
		}

		// 2. UPDATE SERVICES
		services, err := app.FindCollectionByNameOrId("services")
		if err != nil {
			return err
		}
		services.Fields.Add(&core.NumberField{Name: "warranty_months"})
		services.Fields.Add(&core.NumberField{Name: "commission_rate"}) // %
		services.Fields.Add(&core.TextField{Name: "required_skill"})
		if err := app.Save(services); err != nil {
			return err
		}

		// 3. UPDATE TECHNICIANS
		techs, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}
		techs.Fields.Add(&core.SelectField{
			Name:      "level",
			Values:    []string{"junior", "senior", "master"},
			MaxSelect: 1,
		})
		techs.Fields.Add(&core.JSONField{Name: "service_zones"}) // Array of Zone IDs
		techs.Fields.Add(&core.JSONField{Name: "skills"})        // Array of Skill IDs
		techs.Fields.Add(&core.NumberField{Name: "base_salary"})
		techs.Fields.Add(&core.NumberField{Name: "commission_rate"}) // Personal override

		if err := app.Save(techs); err != nil {
			return err
		}

		return nil
	}, nil)
}
