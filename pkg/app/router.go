package app

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"hvac-system/internal/adapter/repository"
	domain "hvac-system/internal/core"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/handlers"
	"hvac-system/pkg/middleware"
	"hvac-system/pkg/services"
	"hvac-system/pkg/ui"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes configures all application routes
func RegisterRoutes(app *pocketbase.PocketBase, t *template.Template, eventBroker *broker.SegmentedBroker, analytics domain.AnalyticsService, bookingServiceInternal domain.BookingService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// [SECURITY] Protect PocketBase Admin UI (/_/)
		// Only allow access if special header is present
		// se.Router.BindFunc(func(e *core.RequestEvent) error {
		// 	if len(e.Request.URL.Path) >= 3 && e.Request.URL.Path[:3] == "/_/" {
		// 		if e.Request.Header.Get("X-Super-Admin") != "mat-khau-cua-toi" {
		// 			return e.String(http.StatusForbidden, "⛔ Super Admin Access Required")
		// 		}
		// 	}
		// 	return e.Next()
		// })

		// ---------------------------------------------------------
		// 1. STATIC FILES & SERVICE WORKERS
		// ---------------------------------------------------------

		// A. Service Worker chính (PWA) - Cho phép scope toàn domain
		se.Router.GET("/assets/service-worker.js", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Service-Worker-Allowed", "/")
			e.Response.Header().Set("Content-Type", "application/javascript")
			return e.FileFS(os.DirFS("./assets"), "service-worker.js")
		})

		// B. Firebase Messaging Service Worker - Phải nằm ở root
		se.Router.GET("/firebase-messaging-sw.js", func(e *core.RequestEvent) error {
			e.Response.Header().Set("Content-Type", "application/javascript")
			return e.FileFS(os.DirFS("./assets"), "firebase-messaging-sw.js")
		})

		// C. Các file tĩnh khác
		se.Router.GET("/assets/{path...}", apis.Static(os.DirFS("./assets"), false))

		// ---------------------------------------------------------
		// 2. SERVICES SETUP
		// ---------------------------------------------------------
		bookingService := services.NewBookingManagementService(app)
		slotService := services.NewTimeSlotService(app)
		inventoryService := services.NewInventoryService(app)
		invoiceService := services.NewInvoiceService(app)

		// --- [QUAN TRỌNG] KHỞI TẠO FCM SERVICE ---
		// Cố gắng load file key, nếu không có thì log cảnh báo chứ không crash app
		fcmService, err := services.NewFCMService("serviceAccountKey.json")
		if err != nil {
			fmt.Printf("⚠️ FCM WARNING: Không tìm thấy hoặc lỗi file 'serviceAccountKey.json': %v\n", err)
			fmt.Println("   Tính năng thông báo đẩy (Push Notification) sẽ KHÔNG hoạt động.")
		} else {
			fmt.Println("✅ FCM Service đã khởi tạo thành công.")
		}

		uiComponents := &ui.Components{
			App:       app,
			Templates: t,
		}

		// Services Check
		techRepo := repository.NewTechnicianRepo(app)
		techService := services.NewTechManagementService(techRepo)

		// [NEW] Settings Repo
		settingsRepo := repository.NewSettingsRepo(app)

		// [NEW] Register Global Middleware for Settings Injection & License Check
		se.Router.BindFunc(middleware.SettingsMiddleware(settingsRepo))

		// ---------------------------------------------------------
		// 3. HANDLERS SETUP
		// ---------------------------------------------------------
		admin := &handlers.AdminHandler{
			App:              app,
			Templates:        t,
			Broker:           eventBroker,
			BookingService:   bookingService,
			SlotService:      slotService,
			TechService:      techService,
			AnalyticsService: analytics,
			UIComponents:     uiComponents,
			SettingsRepo:     settingsRepo, // Injected
		}

		tech := &handlers.TechHandler{
			App:            app,
			Templates:      t,
			Broker:         eventBroker,
			Inventory:      inventoryService,
			InvoiceService: invoiceService,
			BookingService: bookingServiceInternal,
			SettingsRepo:   settingsRepo, // Injected
		}

		slot := &handlers.SlotHandler{
			App:         app,
			SlotService: slotService,
		}

		adminTools := &handlers.AdminToolsHandler{
			App:              app,
			Templates:        t,
			SlotService:      slotService,
			InventoryService: inventoryService,
			Broker:           eventBroker,
		}

		web := &handlers.WebHandler{
			App:          app,
			Templates:    t,
			Broker:       eventBroker,
			SettingsRepo: settingsRepo, // Injected for Public pages
		}

		// --- [MỚI] FCM HANDLER ---
		fcm := &handlers.FCMHandler{
			App:        app,
			FCMService: fcmService,
		}

		public := &handlers.PublicHandler{
			App:            app,
			Templates:      t,
			InvoiceService: invoiceService,
		}

		// ---------------------------------------------------------
		// 4. PUBLIC ROUTES
		// ---------------------------------------------------------
		se.Router.GET("/", public.Index)
		se.Router.GET("/services/{id}", public.ServiceDetail) // [NEW] Detail Page
		se.Router.GET("/book", web.BookingPage)
		se.Router.POST("/book", web.BookService)
		se.Router.GET("/api/slots/available", slot.GetAvailableSlots)

		// Super Invoice Public Routes
		se.Router.GET("/invoice/{hash}", public.ShowInvoice)
		se.Router.POST("/api/invoice/{hash}/feedback", public.SubmitFeedback)

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
		adminGroup.BindFunc(middleware.RequireAdmin(app))

		adminGroup.GET("/", admin.Dashboard)
		adminGroup.GET("/stream", admin.Stream)
		adminGroup.GET("/settings", admin.ShowSettings)
		adminGroup.POST("/settings", admin.UpdateSettings)
		// adminGroup.POST("/bookings/{id}/assign", admin.AssignJob)
		adminGroup.POST("/bookings/{id}/cancel", admin.CancelBooking)
		adminGroup.POST("/bookings/{id}/update", admin.UpdateBookingInfo)
		adminGroup.POST("/bookings/create", admin.CreateBooking) // NEW: Manual Creation
		// adminGroup.POST("/api/bookings/{id}/status", admin.UpdateBookingStatus)

		// Admin Tech Management
		adminGroup.GET("/techs", admin.TechsList)
		adminGroup.POST("/techs/create", admin.CreateTech)
		adminGroup.POST("/techs/{id}/password", admin.ResetTechPassword)
		adminGroup.POST("/techs/{id}/toggle", admin.ToggleTechStatus)

		// Admin Tools
		adminGroup.GET("/tools/slots", adminTools.ShowSlotManager)
		adminGroup.POST("/tools/slots/generate-week", adminTools.GenerateSlotsForWeek)
		adminGroup.GET("/tools/inventory", adminTools.ShowInventoryManager)
		adminGroup.POST("/tools/inventory/create", adminTools.CreateInventoryItem)
		adminGroup.POST("/tools/inventory/{id}/stock", adminTools.UpdateInventoryStock)
		adminGroup.GET("/api/slots", admin.GetSlots)

		// [NEW] Service Management
		adminGroup.GET("/services", admin.ServicesList)
		adminGroup.POST("/services", admin.ServiceSave)
		adminGroup.POST("/services/{id}/delete", admin.ServiceDelete)

		// ---------------------------------------------------------
		// 7. TECH ROUTES (Protected)
		// ---------------------------------------------------------
		techGroup := se.Router.Group("/tech")
		techGroup.BindFunc(middleware.RequireTech(app))

		techGroup.GET("/", func(e *core.RequestEvent) error {
			return e.Redirect(http.StatusSeeOther, "/tech/jobs")
		})
		techGroup.GET("/dashboard", tech.Dashboard)
		techGroup.GET("/jobs", tech.JobsList)
		techGroup.GET("/job/{id}", tech.JobDetail)
		techGroup.GET("/stream", tech.TechStream)
		techGroup.POST("/location", tech.UpdateLocation)
		techGroup.GET("/history", tech.ShowHistory)
		techGroup.GET("/profile", tech.ShowProfile)
		techGroup.POST("/job/{id}/evidence", tech.UploadEvidence)

		// Job Completion Flow
		techGroup.GET("/job/{id}/complete", tech.ShowCompleteJob)
		techGroup.POST("/job/{id}/complete", tech.SubmitCompleteJob)
		techGroup.GET("/job/{id}/invoice-payment", tech.ShowInvoicePayment) // Added missing route

		// Quote & Report Stubs
		techGroup.GET("/jobs/{id}/quote", tech.ShowQuote)
		techGroup.POST("/jobs/{id}/quote", tech.SubmitQuote)
		techGroup.GET("/jobs/{id}/report", tech.ShowReport)
		techGroup.POST("/jobs/{id}/report", tech.SubmitReport)

		// ---------------------------------------------------------
		// 8. TECH API ROUTES (HTMX/Fetch)
		// ---------------------------------------------------------
		apiGroup := se.Router.Group("/api/tech")
		apiGroup.BindFunc(middleware.RequireTech(app))

		// Job Management & Status Updates
		apiGroup.GET("/jobs/list", tech.GetJobsListHTMX)
		apiGroup.POST("/bookings/{id}/status", tech.UpdateJobStatusHTMX) // Fixed route
		apiGroup.POST("/bookings/{id}/cancel", tech.CancelBooking)       // New
		apiGroup.GET("/job/{id}/invoice", tech.GetJobInvoice)
		apiGroup.POST("/job/{id}/payment", tech.ProcessPayment)
		apiGroup.POST("/status/toggle", tech.ToggleOnlineStatus)        // New
		apiGroup.POST("/bookings/{id}/checkin", tech.HandleTechCheckIn) // GPS Check-in
		// Đường dẫn mới sẽ là: /api/tech/fcm/token
		apiGroup.POST("/fcm/token", fcm.RegisterDeviceToken)

		return se.Next()
	})
}
