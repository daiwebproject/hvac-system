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

		// Add 'photo_notes' field
		collection.Fields.Add(&core.TextField{
			Name: "photo_notes",
		})

		// Add 'after_images' field
		collection.Fields.Add(&core.FileField{
			Name:      "after_images",
			MaxSelect: 10,
			MimeTypes: []string{"image/png", "image/jpeg", "image/webp"},
		})

		// Add 'before_images' field (used in tech_handler UploadEvidence)
		collection.Fields.Add(&core.FileField{
			Name:      "before_images",
			MaxSelect: 10,
			MimeTypes: []string{"image/png", "image/jpeg", "image/webp"},
		})

		return app.Save(collection)
	}, nil)
}
