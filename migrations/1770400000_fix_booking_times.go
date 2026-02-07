package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Find bookings with empty booking_time and valid slot
		records, err := app.FindRecordsByFilter(
			"bookings",
			"booking_time = '' && time_slot_id != ''",
			"-created",
			1000,
			0,
			nil,
		)
		if err != nil {
			return err
		}

		fmt.Printf("Found %d bookings to fix...\n", len(records))

		for _, record := range records {
			slotID := record.GetString("time_slot_id")
			slot, err := app.FindRecordById("time_slots", slotID)
			if err != nil {
				fmt.Printf("Skipping booking %s: slot %s not found\n", record.Id, slotID)
				continue
			}

			date := slot.GetString("date")
			startTime := slot.GetString("start_time")

			// Format: YYYY-MM-DD HH:MM
			newTime := fmt.Sprintf("%s %s", date, startTime)
			record.Set("booking_time", newTime)

			if err := app.Save(record); err != nil {
				fmt.Printf("Failed to update booking %s: %v\n", record.Id, err)
			} else {
				fmt.Printf("Fixed booking %s: %s\n", record.Id, newTime)
			}
		}
		return nil
	}, func(app core.App) error {
		return nil
	})
}
