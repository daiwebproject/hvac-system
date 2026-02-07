package app

import (
	"net/http"
	"os"

	internalApp "hvac-system/internal/app"
	"hvac-system/pkg/handlers"
	"hvac-system/pkg/middleware"
	"hvac-system/pkg/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes configures all application routes
// Now accepts a single Container instead of 10+ individual parameters
func RegisterRoutes(pb *pocketbase.PocketBase, c *internalApp.Container) {
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// ---------------------------------------------------------
		// 1. STATIC FILES & SERVICE WORKERS
		// ---------------------------------------------------------

		// A. Service Worker chính (PWA) - Cho phép scope toàn domain
		se.Router.GET("/service-worker.js", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Service-Worker-Allowed", "/")
			e.Response.Header().Set("Content-Type", "application/javascript")
			return e.FileFS(os.DirFS("."), "service-worker.js")
		})

		// [ALIAS] Legacy SW path
		se.Router.GET("/assets/service-worker.js", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Service-Worker-Allowed", "/")
			e.Response.Header().Set("Content-Type", "application/javascript")
			return e.FileFS(os.DirFS("."), "service-worker.js")
		})

		// B. Service Main Manifests (from root)
		se.Router.GET("/manifest.json", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Content-Type", "application/json")
			e.Response.Header().Set("Cache-Control", "no-cache")
			return e.FileFS(os.DirFS("./assets"), "manifest.json")
		})

		se.Router.GET("/manifest-admin.json", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Content-Type", "application/json")
			e.Response.Header().Set("Cache-Control", "no-cache")
			return e.FileFS(os.DirFS("./assets"), "manifest-admin.json")
		})

		// C. Firebase Messaging Service Worker - Phải nằm ở root
		se.Router.GET("/firebase-messaging-sw.js", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Content-Type", "application/javascript")
			return e.FileFS(os.DirFS("./assets"), "firebase-messaging-sw.js")
		})

		// D. Các file tĩnh khác - Catch all assets
		se.Router.GET("/assets/{path...}", apis.Static(os.DirFS("./assets"), false))

		// ---------------------------------------------------------
		// 2. SERVICES FROM CONTAINER (No more local initialization)
		// ---------------------------------------------------------
		// Legacy pkg/services that still need PocketBase directly
		slotService := services.NewTimeSlotService(pb, c.TechRepo, c.BookingRepo)

		// Register Global Middleware for Settings Injection & License Check
		se.Router.BindFunc(middleware.SettingsMiddleware(c.SettingsRepo))

		// ---------------------------------------------------------
		// 3. HANDLERS SETUP (Using Container dependencies)
		// ---------------------------------------------------------
		admin := &handlers.AdminHandler{
			App:              pb,
			Templates:        c.Templates,
			Broker:           c.Broker,
			BookingService:   c.BookingService,
			SlotService:      slotService,
			TechService:      c.TechService,
			AnalyticsService: c.AnalyticsService,
			UIComponents:     c.UIComponents,
			SettingsRepo:     c.SettingsRepo,
			FCMService:       c.FCMService,
		}

		tech := &handlers.TechHandler{
			App:            pb,
			Templates:      c.Templates,
			Broker:         c.Broker,
			Inventory:      c.InventoryService,
			InvoiceService: c.InvoiceService,
			BookingService: c.BookingService,
			SettingsRepo:   c.SettingsRepo,
			FCMService:     c.FCMService,
			TechRepo:       c.TechRepo,
			BookingRepo:    c.BookingRepo,
		}

		slot := &handlers.SlotHandler{
			App:         pb,
			SlotService: slotService,
		}

		adminTools := &handlers.AdminToolsHandler{
			App:              pb,
			Templates:        c.Templates,
			SlotService:      slotService,
			InventoryService: c.InventoryService,
			Broker:           c.Broker,
		}

		web := &handlers.WebHandler{
			App:            pb,
			Templates:      c.Templates,
			Broker:         c.Broker,
			SettingsRepo:   c.SettingsRepo,
			FCMService:     c.FCMService,
			BookingService: c.BookingService,
			SlotService:    slotService,
		}

		fcm := &handlers.FCMHandler{
			App:          pb,
			FCMService:   c.FCMService,
			SettingsRepo: c.SettingsRepo,
			TechRepo:     c.TechRepo,
		}

		public := &handlers.PublicHandler{
			App:            pb,
			Templates:      c.Templates,
			InvoiceService: c.InvoiceService,
		}

		// Location handlers from Container
		locationHandler := c.LocationHandler
		locationSSEHandler := c.LocationSSEHandler

		// ---------------------------------------------------------
		// 4. PUBLIC ROUTES
		// ---------------------------------------------------------
		se.Router.GET("/", public.Index)
		se.Router.GET("/services/{id}", public.ServiceDetail)
		se.Router.GET("/book", web.BookingPage)
		se.Router.POST("/book", web.BookService)
		se.Router.GET("/api/slots/available", slot.GetAvailableSlots)
		se.Router.GET("/api/public/reverse-geocode", public.ReverseGeocode)
		se.Router.GET("/api/public/geocode", public.Geocode)

		// Super Invoice Public Routes
		se.Router.GET("/invoice/{hash}", public.ShowInvoice)
		se.Router.POST("/api/invoice/{hash}/feedback", public.SubmitFeedback)

		// ----- LOCATION TRACKING - PUBLIC ROUTES -----
		se.Router.GET("/api/health/location", locationHandler.HealthCheck)
		se.Router.GET("/api/bookings/{id}/tech-location", locationHandler.GetBookingTechLocation)
		se.Router.GET("/api/bookings/{id}/location/stream", locationSSEHandler.StreamCustomerLocation)
		se.Router.GET("/api/tech/{id}/events/stream", locationSSEHandler.StreamTechnicianEvents)

		// ---------------------------------------------------------
		// 5. AUTH ROUTES
		// ---------------------------------------------------------
		se.Router.GET("/login", admin.ShowLogin)
		se.Router.POST("/login", admin.ProcessLogin)
		se.Router.GET("/admin/logout", admin.Logout)

		se.Router.GET("/tech/login", tech.ShowLogin)
		se.Router.POST("/tech/login", tech.ProcessLogin)
		se.Router.GET("/tech/logout", tech.Logout)

		// ---------------------------------------------------------
		// 6. ADMIN ROUTES (Protected)
		// ---------------------------------------------------------
		adminGroup := se.Router.Group("/admin")
		adminGroup.BindFunc(middleware.RequireAdmin(pb))

		adminGroup.GET("/", admin.Dashboard)
		adminGroup.GET("/stream", admin.Stream)
		adminGroup.GET("/settings", admin.ShowSettings)
		adminGroup.POST("/settings", admin.UpdateSettings)

		// History page and search API
		adminGroup.GET("/history", admin.ShowHistory)
		adminGroup.GET("/api/bookings/search", admin.SearchHistory)

		adminGroup.POST("/bookings/{id}/assign", admin.AssignJob)
		adminGroup.POST("/bookings/{id}/cancel", admin.CancelBooking)
		adminGroup.POST("/bookings/{id}/update", admin.UpdateBookingInfo)
		adminGroup.POST("/bookings/create", admin.CreateBooking)
		adminGroup.POST("/api/bookings/{id}/status", admin.UpdateBookingStatus)
		// [NEW] API for fetching active bookings for conflict check
		adminGroup.GET("/api/bookings/active", admin.ActiveBookings)

		// Admin Tech Management
		adminGroup.GET("/techs", admin.TechsList)
		adminGroup.POST("/techs/create", admin.CreateTech)
		adminGroup.POST("/techs/{id}/password", admin.ResetTechPassword)
		adminGroup.POST("/techs/{id}/toggle", admin.ToggleTechStatus)

		// FCM Token
		adminGroup.POST("/fcm/token", fcm.RegisterDeviceToken)
		adminGroup.GET("/debug/fcm-tokens", admin.DebugAdminTokens)

		// Admin Tools
		adminGroup.GET("/tools/slots", adminTools.ShowSlotManager)
		adminGroup.POST("/tools/slots/generate-week", adminTools.GenerateSlotsForWeek)
		adminGroup.GET("/tools/inventory", adminTools.ShowInventoryManager)
		adminGroup.POST("/tools/inventory/create", adminTools.CreateInventoryItem)
		adminGroup.POST("/tools/inventory/{id}/stock", adminTools.UpdateInventoryStock)
		adminGroup.GET("/api/slots", admin.GetSlots)

		// Service Management
		adminGroup.GET("/services", admin.ServicesList)
		adminGroup.POST("/services", admin.ServiceSave)
		adminGroup.POST("/services/{id}/delete", admin.ServiceDelete)

		// Category Management
		adminGroup.GET("/categories", admin.CategoriesList)
		adminGroup.POST("/categories", admin.CategorySave)
		adminGroup.POST("/categories/{id}/delete", admin.CategoryDelete)

		// ----- LOCATION TRACKING - ADMIN ROUTES -----
		adminGroup.GET("/api/locations", locationHandler.GetAllTechLocations)
		adminGroup.GET("/api/locations/stream", locationSSEHandler.StreamAdminLocations)

		// ---------------------------------------------------------
		// 7. TECH ROUTES (Trang giao diện chính)
		// ---------------------------------------------------------
		techGroup := se.Router.Group("/tech")
		techGroup.BindFunc(middleware.RequireTech(pb))

		techGroup.GET("/", func(e *core.RequestEvent) error {
			return e.Redirect(http.StatusSeeOther, "/tech/dashboard")
		})
		techGroup.GET("/dashboard", tech.Dashboard)
		techGroup.GET("/jobs", tech.JobsList)
		techGroup.GET("/job/{id}", tech.JobDetail)
		techGroup.GET("/history", tech.ShowHistory)
		techGroup.GET("/profile", tech.ShowProfile)
		techGroup.GET("/stream", tech.TechStream)

		// Luồng hoàn thành công việc
		techGroup.GET("/job/{id}/complete", tech.ShowCompleteJob)
		techGroup.POST("/job/{id}/complete", tech.SubmitCompleteJob)
		techGroup.GET("/job/{id}/invoice-payment", tech.ShowInvoicePayment)

		// ---------------------------------------------------------
		// 8. TECH API ROUTES (Dành cho HTMX và Xử lý dữ liệu)
		// ---------------------------------------------------------
		apiGroup := se.Router.Group("/api/tech")
		apiGroup.BindFunc(middleware.RequireTech(pb))

		apiGroup.GET("/jobs/list", tech.JobsList)
		apiGroup.GET("/schedule", tech.GetSchedule)

		// Quản lý trạng thái và thao tác
		apiGroup.POST("/status/toggle", tech.ToggleOnlineStatus)
		apiGroup.POST("/bookings/{id}/checkin", tech.HandleTechCheckIn)
		apiGroup.POST("/bookings/{id}/status", tech.UpdateJobStatusHTMX)
		apiGroup.POST("/bookings/{id}/cancel", tech.CancelBooking)
		apiGroup.POST("/location", tech.UpdateLocation)
		apiGroup.POST("/fcm/token", fcm.RegisterDeviceToken)

		// ----- LOCATION TRACKING API -----
		apiGroup.POST("/location/update", locationHandler.UpdateLocation)
		apiGroup.POST("/tracking/start", locationHandler.StartTracking)
		apiGroup.POST("/tracking/stop", locationHandler.StopTracking)
		apiGroup.GET("/location", locationHandler.GetTechLocation)

		// Hóa đơn và thanh toán
		apiGroup.GET("/job/{id}/invoice", tech.GetJobInvoice)
		apiGroup.POST("/job/{id}/payment", tech.ProcessPayment)

		return se.Next()
	})
}
