package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}

		// Check if tech exists
		existing, _ := app.FindAuthRecordByEmail("technicians", "tech@demo.com")
		if existing != nil {
			return nil
		}

		record := core.NewRecord(collection)
		record.SetEmail("tech@demo.com")
		record.SetPassword("12345678")
		record.Set("name", "Kỹ thuật viên Demo")
		record.Set("phone", "0909000111")
		record.Set("skills", "AC, Washer")
		record.SetVerified(true)

		return app.Save(record)
	}, func(app core.App) error {
		return nil
	})
}
