package main

import (
	"log"

	"hvac-system/pkg/app"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/services"

	_ "hvac-system/migrations"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func main() {
	pb := pocketbase.New()

	// 1. Migrations
	migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	// 2. Event Broker (Segmented)
	eventBroker := broker.NewSegmentedBroker()

	// 3. App Initialization
	templates, err := app.InitTemplates()
	if err != nil {
		log.Fatal("Error initializing templates:", err)
	}

	// 3.5 Services
	analyticsService := services.NewAnalyticsService(pb)

	// 4. Register Routes
	app.RegisterRoutes(pb, templates, eventBroker, analyticsService)

	// 5. Hooks (TODO: Implement with segmented broker)
	// uiComponents := &ui.Components{
	// 	App:       pb,
	// 	Templates: templates,
	// }

	// TODO: Implement proper event publishing with segmented broker
	// pb.OnRecordAfterCreateSuccess("bookings").BindFunc(func(re *core.RecordEvent) error {
	// 	htmlRow, err := uiComponents.RenderBookingRow(re.Record)
	// 	if err == nil {
	// 		eventBroker.Publish(broker.ChannelAdmin, "", broker.Event{
	// 			Type:      "booking.created",
	// 			Timestamp: time.Now().Unix(),
	// 			Data:      map[string]interface{}{"html": htmlRow},
	// 		})
	// 	}
	// 	return re.Next()
	// })

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}

