package main

import (
	"log"

	"hvac-system/internal/adapter/repository"
	"hvac-system/internal/handler"
	"hvac-system/internal/service"
	"hvac-system/pkg/app"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/middleware"

	_ "hvac-system/migrations"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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

	// Domain Services
	slotService := service.NewTimeSlotService(slotRepo, bookingRepo, svcRepo)
	analyticsServiceInternal := service.NewAnalyticsService(analyticsRepo)

	// Booking Service
	bookingServiceInternal := service.NewBookingService(bookingRepo, techRepo, slotService)

	// Handlers
	bookingHandler := handler.NewBookingHandler(bookingServiceInternal)

	// 4. Register Routes (Legacy)
	app.RegisterRoutes(pb, templates, eventBroker, analyticsServiceInternal, bookingServiceInternal)

	// Register New Handler Routes (Mixing with legacy for transition)
	pb.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Admin Group Extensions
		adminGroup := e.Router.Group("/admin")
		adminGroup.BindFunc(middleware.RequireAdmin(pb))

		// Booking Routes
		adminGroup.POST("/bookings/{id}/assign", bookingHandler.AssignJob)
		adminGroup.POST("/api/bookings/{id}/status", bookingHandler.UpdateStatus)

		return e.Next()
	})

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}
