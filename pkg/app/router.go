package app

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

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
func RegisterRoutes(app *pocketbase.PocketBase, t *template.Template, eventBroker *broker.SegmentedBroker, analytics *services.AnalyticsService) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

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

		// ---------------------------------------------------------
		// 3. HANDLERS SETUP
		// ---------------------------------------------------------
		admin := &handlers.AdminHandler{
			App:              app,
			Templates:        t,
			Broker:           eventBroker,
			BookingService:   bookingService,
			SlotService:      slotService,
			AnalyticsService: analytics,
			UIComponents:     uiComponents,
		}

		tech := &handlers.TechHandler{
			App:            app,
			Templates:      t,
			Broker:         eventBroker,
			Inventory:      inventoryService,
			InvoiceService: invoiceService,
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
			App:       app,
			Templates: t,
			Broker:    eventBroker,
		}

		// --- [MỚI] FCM HANDLER ---
		fcm := &handlers.FCMHandler{
			App:        app,
			FCMService: fcmService,
		}

		// ---------------------------------------------------------
		// 4. PUBLIC ROUTES
		// ---------------------------------------------------------
		se.Router.GET("/", web.Index)
		se.Router.GET("/book", web.BookingPage)
		se.Router.POST("/book", web.BookService)
		se.Router.GET("/api/slots/available", slot.GetAvailableSlots)

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
		adminGroup.POST("/bookings/{id}/assign", admin.AssignJob)
		adminGroup.POST("/bookings/{id}/cancel", admin.CancelBooking)
		adminGroup.POST("/bookings/{id}/update", admin.UpdateBookingInfo)
		adminGroup.POST("/api/bookings/{id}/status", admin.UpdateBookingStatus)

		// Admin Tools
		adminGroup.GET("/tools/slots", adminTools.ShowSlotManager)
		adminGroup.POST("/tools/slots/generate-week", adminTools.GenerateSlotsForWeek)
		adminGroup.GET("/tools/inventory", adminTools.ShowInventoryManager)
		adminGroup.POST("/tools/inventory/create", adminTools.CreateInventoryItem)
		adminGroup.POST("/tools/inventory/{id}/stock", adminTools.UpdateInventoryStock)
		adminGroup.GET("/api/slots", admin.GetSlots)

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

		apiGroup.POST("/job/{id}/status", tech.UpdateJobStatusHTMX)
		apiGroup.GET("/job/{id}/invoice", tech.GetJobInvoice)
		apiGroup.POST("/job/{id}/payment", tech.ProcessPayment)
		apiGroup.GET("/jobs/list", tech.GetJobsListHTMX)
		// Đường dẫn mới sẽ là: /api/tech/fcm/token
		apiGroup.POST("/fcm/token", fcm.RegisterDeviceToken)

		return se.Next()
	})
}
