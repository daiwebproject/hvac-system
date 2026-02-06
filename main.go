package main

import (
	"fmt"
	"log"

	"hvac-system/internal/adapter/repository"
	"hvac-system/internal/handler"
	"hvac-system/internal/service"
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

	// --- [NEW ARCHITECTURE WIRING] ---
	// Adapters
	bookingRepo := repository.NewBookingRepo(pb)
	techRepo := repository.NewTechnicianRepo(pb)
	slotRepo := repository.NewTimeSlotRepo(pb)
	svcRepo := repository.NewServiceRepo(pb)
	analyticsRepo := repository.NewAnalyticsRepo(pb)
	settingsRepo := repository.NewSettingsRepo(pb) // [NEW]

	// Domain Services
	slotService := service.NewTimeSlotService(slotRepo, bookingRepo, svcRepo)
	analyticsServiceInternal := service.NewAnalyticsService(analyticsRepo)

	// [NEW] Location Cache for Real-time Tracking
	locationCache := services.NewLocationCache()

	// [NEW] FCM Service
	fcmService, err := services.NewFCMService("serviceAccountKey.json")
	if err != nil {
		fmt.Printf("⚠️ FCM WARNING: %v\n", err)
	} else {
		fmt.Println("✅ FCM Service Initialized")
	}

	// Booking Service (injected with FCM and Broker)
	bookingServiceInternal := service.NewBookingService(bookingRepo, techRepo, slotService, slotRepo, fcmService, settingsRepo, eventBroker)

	// Handlers
	locationHandler := handler.NewLocationHandler(locationCache, bookingRepo, techRepo, eventBroker)
	locationSSEHandler := handler.NewLocationSSEHandler(eventBroker)

	// 4. Register Routes (Legacy)
	app.RegisterRoutes(pb, templates, eventBroker, analyticsServiceInternal, bookingServiceInternal, fcmService, locationCache, locationHandler, locationSSEHandler)

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}
