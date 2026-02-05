package main

import (
	"fmt"
	"log"

	"hvac-system/internal/adapter/repository"
	"hvac-system/internal/handler"
	"hvac-system/internal/service"
	"hvac-system/pkg/app"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/middleware"
	"hvac-system/pkg/services"

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
	settingsRepo := repository.NewSettingsRepo(pb) // [NEW]

	// Domain Services
	slotService := service.NewTimeSlotService(slotRepo, bookingRepo, svcRepo)
	analyticsServiceInternal := service.NewAnalyticsService(analyticsRepo)

	// [NEW] FCM Service
	fcmService, err := services.NewFCMService("serviceAccountKey.json")
	if err != nil {
		fmt.Printf("⚠️ FCM WARNING: %v\n", err)
	} else {
		fmt.Println("✅ FCM Service Initialized")
	}

	// Booking Service (injected with FCM)
	// Booking Service (injected with FCM and Broker)
	bookingServiceInternal := service.NewBookingService(bookingRepo, techRepo, slotService, fcmService, settingsRepo, eventBroker)

	// Handlers
	bookingHandler := handler.NewBookingHandler(bookingServiceInternal)

	// 4. Register Routes (Legacy)
	app.RegisterRoutes(pb, templates, eventBroker, analyticsServiceInternal, bookingServiceInternal, fcmService)

	// Register New Handler Routes (Mixing with legacy for transition)
	pb.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Admin Group Extensions
		adminGroup := e.Router.Group("/admin")
		adminGroup.BindFunc(middleware.RequireAdmin(pb))

		// Booking Routes
		adminGroup.POST("/bookings/{id}/assign", bookingHandler.AssignJob)
		adminGroup.POST("/api/bookings/{id}/status", bookingHandler.UpdateStatus)

		// [NEW] Admin FCM Token Registration
		adminGroup.POST("/fcm/token", func(e *core.RequestEvent) error {
			// Using the existing AdminHandler instance managed by app package is tricky here because
			// 'adminHandler' variable here is 'bookingHandler' (which is handler.NewBookingHandler).
			// The Legacy AdminHandler is likely initialized inside app.RegisterRoutes.
			// Ideally we should move this route to app.RegisterRoutes.
			// But for quick fix, we can re-instantiate or use the main one if exported.

			// Actually, app.RegisterRoutes likely registers admin routes too.
			// Let's check app.RegisterRoutes signature.
			// It returns nothing.

			// We can delegate this logic to a simple inline func or look at how AdminHandler is used.
			// The prompt says "Register New Handler Routes".
			// We need an AdminHandler instance here ONLY IF we want to use its methods.
			// But we have access to 'settingsRepo' variable in main!
			// We can just write the handler func inline using the closure capabilities of Go.

			type TokenRequest struct {
				Token string `json:"token"`
			}
			var req TokenRequest
			if err := e.BindBody(&req); err != nil {
				return e.String(400, "Invalid JSON")
			}
			if req.Token == "" {
				return e.String(400, "Token required")
			}
			fmt.Printf("👉 [ADMIN_INLINE] Received FCM Token: %s\n", req.Token)
			if err := settingsRepo.AddAdminToken(req.Token); err != nil {
				return e.String(500, "Failed to save token: "+err.Error())
			}
			return e.JSON(200, map[string]string{"message": "Token registered"})
		})

		return e.Next()
	})

	if err := pb.Start(); err != nil {
		log.Fatal(err)
	}
}
