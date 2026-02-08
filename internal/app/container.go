// Package app provides the dependency injection container for the HVAC system.
// This consolidates all service initialization in one place.
package app

import (
	"fmt"
	"html/template"

	"hvac-system/internal/adapter/repository"
	domain "hvac-system/internal/core"
	"hvac-system/internal/handler"
	"hvac-system/internal/service"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/cache"
	"hvac-system/pkg/notification"
	"hvac-system/pkg/services"
	"hvac-system/pkg/ui"

	"github.com/pocketbase/pocketbase"
)

// Container holds all application dependencies.
// This is the central place for Dependency Injection.
type Container struct {
	// PocketBase instance
	PB *pocketbase.PocketBase

	// Templates
	Templates *template.Template

	// Infrastructure
	Broker *broker.SegmentedBroker

	// Repositories (Data Access Layer)
	BookingRepo   domain.BookingRepository
	TechRepo      domain.TechnicianRepository
	SlotRepo      domain.TimeSlotRepository
	ServiceRepo   domain.ServiceRepository
	AnalyticsRepo domain.AnalyticsRepository
	SettingsRepo  *repository.SettingsRepo // Concrete type for handler compatibility
	BrandRepo     domain.BrandRepository   // [NEW] SaaS Brand Management

	// Domain Services (Business Logic)
	BookingService   domain.BookingService
	SlotService      domain.TimeSlotControl // Interface for slot operations
	AnalyticsService domain.AnalyticsService
	TechService      *services.TechManagementService
	InventoryService *services.InventoryService
	InvoiceService   *services.InvoiceService

	// External Services (New package locations)
	FCMService    *notification.FCMService
	LocationCache *cache.LocationCache

	// Handlers (internal package)
	LocationHandler    *handler.LocationHandler
	LocationSSEHandler *handler.LocationSSEHandler

	// UI Components
	UIComponents *ui.Components
}

// NewContainer creates and wires all dependencies.
// This replaces the manual wiring in main.go.
func NewContainer(pb *pocketbase.PocketBase) (*Container, error) {
	c := &Container{
		PB: pb,
	}

	// 1. Event Broker
	c.Broker = broker.NewSegmentedBroker()

	// 2. Templates (from internal/app/templates.go)
	templates, err := InitTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to init templates: %w", err)
	}
	c.Templates = templates

	// 3. Repositories (Adapters)
	c.BookingRepo = repository.NewBookingRepo(pb)
	c.TechRepo = repository.NewTechnicianRepo(pb)
	c.SlotRepo = repository.NewTimeSlotRepo(pb)
	c.ServiceRepo = repository.NewServiceRepo(pb)
	c.AnalyticsRepo = repository.NewAnalyticsRepo(pb)
	c.SettingsRepo = repository.NewSettingsRepo(pb)
	c.BrandRepo = repository.NewBrandRepo(pb)

	// 4. External Services (from new packages)
	c.LocationCache = cache.NewLocationCache()

	fcmService, err := notification.NewFCMService("serviceAccountKey.json")
	if err != nil {
		fmt.Printf("⚠️ FCM WARNING: %v\n", err)
		// FCM is optional, continue without it
	} else {
		fmt.Println("✅ FCM Service Initialized")
	}
	c.FCMService = fcmService

	// 5. Domain Services (inject repos + external services)
	c.SlotService = service.NewTimeSlotService(c.SlotRepo, c.BookingRepo, c.ServiceRepo)
	c.AnalyticsService = service.NewAnalyticsService(c.AnalyticsRepo)
	c.BookingService = service.NewBookingService(
		c.BookingRepo,
		c.TechRepo,
		c.SlotService,
		c.SlotRepo,
		c.FCMService,
		c.SettingsRepo,
		c.Broker,
	)

	// 6. pkg/services (legacy, will be migrated in future phases)
	c.TechService = services.NewTechManagementService(c.TechRepo)
	c.InventoryService = services.NewInventoryService(pb)
	c.InvoiceService = services.NewInvoiceService(pb)

	// 7. Internal Handlers
	c.LocationHandler = handler.NewLocationHandler(c.LocationCache, c.BookingRepo, c.TechRepo, c.Broker)
	c.LocationSSEHandler = handler.NewLocationSSEHandler(c.Broker)

	// 8. UI Components
	c.UIComponents = &ui.Components{
		App:       pb,
		Templates: c.Templates,
	}

	return c, nil
}
