package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jobReports, err := app.FindCollectionByNameOrId("job_reports")
		if err != nil {
			return err
		}

		// Add before_images field (photos taken at job start)
		if jobReports.Fields.GetByName("before_images") == nil {
			jobReports.Fields.Add(&core.FileField{
				Name:      "before_images",
				MaxSelect: 5,
				MaxSize:   10485760, // 10MB per file
			})
		}

		// Add after_images field (photos taken at completion, REQUIRED)
		if jobReports.Fields.GetByName("after_images") == nil {
			jobReports.Fields.Add(&core.FileField{
				Name:      "after_images",
				Required:  true, // Mandatory for job completion
				MaxSelect: 5,
				MaxSize:   10485760, // 10MB per file
			})
		}

		// Add photo_notes field for technician to describe the work done
		if jobReports.Fields.GetByName("photo_notes") == nil {
			jobReports.Fields.Add(&core.TextField{
				Name: "photo_notes",
			})
		}

		return app.Save(jobReports)

	}, func(app core.App) error {
		// Rollback: Fields will remain but can be manually removed if needed
		// PocketBase doesn't provide a simple Remove method for fields
		return nil
	})
}
