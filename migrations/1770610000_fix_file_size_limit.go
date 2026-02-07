package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("job_reports")
		if err != nil {
			return err
		}

		// Fix 'before_images' - Set MaxSize to 10MB
		if field := collection.Fields.GetByName("before_images"); field != nil {
			if f, ok := field.(*core.FileField); ok {
				f.MaxSize = 10485760 // 10MB
				f.MaxSelect = 10
			}
		}

		// Fix 'after_images' - Set MaxSize to 10MB
		if field := collection.Fields.GetByName("after_images"); field != nil {
			if f, ok := field.(*core.FileField); ok {
				f.MaxSize = 10485760 // 10MB
				f.MaxSelect = 10
			}
		}

		// Fix 'proof_images' (old field) - Set MaxSize to 10MB
		if field := collection.Fields.GetByName("proof_images"); field != nil {
			if f, ok := field.(*core.FileField); ok {
				f.MaxSize = 10485760 // 10MB
				f.MaxSelect = 10
			}
		}

		return app.Save(collection)
	}, nil)
}
