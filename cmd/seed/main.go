package main

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		collection, err := app.FindCollectionByNameOrId("services")
		if err != nil {
			return err
		}

		// Check if service exists
		existing, _ := app.FindRecordsByFilter("services", "name='Maintenance Service'", "", 1, 0, nil)
		if len(existing) > 0 {
			fmt.Printf("Service already exists: %s\n", existing[0].Id)
			return nil
		}

		record := core.NewRecord(collection)
		record.Set("name", "Maintenance Service")
		record.Set("price", 500000)
		record.Set("active", true)
		record.Set("description", "General maintenance")

		if err := app.Save(record); err != nil {
			return err
		}

		fmt.Printf("Created service: %s\n", record.Id)
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
