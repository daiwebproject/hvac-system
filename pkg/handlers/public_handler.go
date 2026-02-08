package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	domain "hvac-system/internal/core"
	"hvac-system/pkg/services"

	"github.com/pocketbase/pocketbase/core"
)

type PublicHandler struct {
	App            core.App
	Templates      *template.Template
	InvoiceService *services.InvoiceService
	BrandRepo      domain.BrandRepository // [NEW] Link to Brand
}

// Index renders the homepage with dynamic data
// GET /
func (h *PublicHandler) Index(e *core.RequestEvent) error {
	// 1. Get active Services
	services, _ := h.App.FindRecordsByFilter("services", "active=true", "-created", 100, 0, nil)

	// 2. Get Brand (SaaS)
	brand, err := h.BrandRepo.GetDefault()
	if err != nil {
		fmt.Printf("⚠️ Default brand not found: %v\n", err)
	}

	// 3. Prepare Data
	data := map[string]interface{}{
		"Services": services,
		"Brand":    brand, // New standard
		// Compatibility aliases for templates
		"Settings":     brand,
		"PageSettings": brand,
	}

	return RenderPage(h.Templates, e, "layouts/base.html", "public/index.html", data)
}

// ShowInvoice renders the public invoice view
// GET /invoice/{hash}
func (h *PublicHandler) ShowInvoice(e *core.RequestEvent) error {
	hash := e.Request.PathValue("hash")
	if hash == "" {
		return e.String(404, "Invoice not found")
	}

	// 1. Find Invoice by Hash
	invoices, err := h.App.FindRecordsByFilter("invoices", fmt.Sprintf("public_hash='%s'", hash), "", 1, 0, nil)
	if err != nil || len(invoices) == 0 {
		return e.String(404, "Invoice not found or link expired")
	}
	invoice := invoices[0]
	bookingID := invoice.GetString("booking_id")

	// 2. Fetch Booking
	booking, err := h.App.FindRecordById("bookings", bookingID)
	if err != nil {
		return e.String(500, "Booking data mismatched")
	}

	var report *core.Record
	reports, _ := h.App.FindRecordsByFilter("job_reports", fmt.Sprintf("booking_id='%s'", bookingID), "-created", 1, 0, nil)
	if len(reports) > 0 {
		report = reports[0]
		fmt.Printf("DEBUG INVOICEVIEW: Found Report ID=%s, Photos=%d, Notes=%s\n",
			report.Id,
			len(report.GetStringSlice("after_images")),
			report.GetString("photo_notes"))
	} else {
		fmt.Printf("DEBUG INVOICEVIEW: No Report found for booking %s\n", bookingID)
	}

	// 4. Fetch Job Parts (Materials)
	items, _ := h.App.FindRecordsByFilter("invoice_items", fmt.Sprintf("invoice_id='%s'", invoice.Id), "", 100, 0, nil)

	// 5. Fetch Brand (SaaS)
	brand, _ := h.BrandRepo.GetDefault()

	// 6. Fetch Technician (Keeping existing logic)
	techID := booking.GetString("technician_id")
	tech, _ := h.App.FindRecordById("technicians", techID)

	data := map[string]interface{}{
		"Invoice":  invoice,
		"Job":      booking,
		"Tech":     tech,
		"Parts":    items, // Mapping 'Items' to 'Parts' in template or vice versa. Template uses .Items
		"Items":    items,
		"Report":   report,
		"Settings": brand, // Keep legacy key for layout compatibility
		"Brand":    brand,
	}

	return RenderPage(h.Templates, e, "invoice_view", "public/invoice_view.html", data)
}

// SubmitFeedback handles customer signature and rating
// POST /api/invoice/{hash}/feedback
func (h *PublicHandler) SubmitFeedback(e *core.RequestEvent) error {
	hash := e.Request.PathValue("hash")

	invoices, err := h.App.FindRecordsByFilter("invoices", fmt.Sprintf("public_hash='%s'", hash), "", 1, 0, nil)
	if err != nil || len(invoices) == 0 {
		return e.JSON(404, map[string]string{"error": "Invoice not found"})
	}
	invoice := invoices[0]
	bookingID := invoice.GetString("booking_id")

	// Get form data
	signatureBase64 := e.Request.FormValue("signature") // Data URL
	rating := e.Request.FormValue("rating")

	if signatureBase64 == "" {
		return e.JSON(400, map[string]string{"error": "Signature required"})
	}

	// Save to Booking (Signature) and Invoice (Rating)
	// 1. Update Booking with Signature
	booking, err := h.App.FindRecordById("bookings", bookingID)
	if err == nil {
		booking.Set("customer_rating", rating)
		booking.Set("signed_at", time.Now())
		h.App.Save(booking)
	}

	// Update Invoice status?
	invoice.Set("status", "signed") // If we have such status
	h.App.Save(invoice)

	return e.JSON(200, map[string]string{"success": "true"})
}

// GET /services/{id}
func (h *PublicHandler) ServiceDetail(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	service, err := h.App.FindRecordById("services", id)
	if err != nil {
		return e.String(404, "Dịch vụ không tồn tại")
	}

	// Fetch unrelated services for "Other Services" section?
	// or just active services
	otherServices, _ := h.App.FindRecordsByFilter("services", "active=true && id != '"+id+"'", "-created", 4, 0, nil)

	// Convert HTML content to template.HTML for safe rendering
	detailContent := service.GetString("detail_content")

	// Fetch Brand for layout
	brand, _ := h.BrandRepo.GetDefault()

	data := map[string]interface{}{
		"Service":           service,
		"OtherServices":     otherServices,
		"DetailContentHTML": template.HTML(detailContent),
		"Settings":          brand, // For base.html
		"Brand":             brand,
	}

	return RenderPage(h.Templates, e, "layouts/base.html", "public/service_detail.html", data)
}

// ReverseGeocode proxies requests to Nominatim to avoid CORS issues
// GET /api/public/reverse-geocode?lat=...&lon=...
func (h *PublicHandler) ReverseGeocode(e *core.RequestEvent) error {
	lat := e.Request.URL.Query().Get("lat")
	lon := e.Request.URL.Query().Get("lon")

	if lat == "" || lon == "" {
		return e.JSON(400, map[string]string{"error": "Missing lat/lon parameters"})
	}

	// Construct Nominatim URL
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%s&lon=%s&zoom=18&addressdetails=1", lat, lon)

	// Create Request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return e.JSON(500, map[string]string{"error": "Failed to create request"})
	}

	// IMPORTANT: Set User-Agent as required by Nominatim usage policy
	// Avoid "example.com" as it is often blocked.
	req.Header.Set("User-Agent", "HVAC-Service/1.0")
	req.Header.Set("Referer", "https://hvac-system.local")

	// Execute Request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ [REVERSE_GEO] Failed to contact Nominatim: %v\n", err)
		return e.JSON(502, map[string]string{"error": "Failed to contact Nominatim"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ [REVERSE_GEO] Nominatim returned status: %d\n", resp.StatusCode)
		// Try to read body for error details
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		fmt.Printf("❌ [REVERSE_GEO] Nominatim Body: %+v\n", body)

		return e.JSON(resp.StatusCode, map[string]string{"error": "Nominatim error"})
	}

	// Decode and Proxy Response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("❌ [REVERSE_GEO] Invalid JSON: %v\n", err)
		return e.JSON(500, map[string]string{"error": "Invalid JSON from Nominatim"})
	}

	fmt.Println("✅ [REVERSE_GEO] Success")
	return e.JSON(200, result)
}

// Geocode proxies forward geocoding requests to Nominatim
// GET /api/public/geocode?q=...
func (h *PublicHandler) Geocode(e *core.RequestEvent) error {
	query := e.Request.URL.Query().Get("q")
	if query == "" {
		return e.JSON(400, map[string]string{"error": "Missing query parameter"})
	}

	// Construct Nominatim URL - Query MUST be URL-encoded for special characters
	encodedQuery := url.QueryEscape(query)
	nominatimURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?format=json&q=%s", encodedQuery)

	// Create Request
	req, err := http.NewRequest("GET", nominatimURL, nil)
	if err != nil {
		return e.JSON(500, map[string]string{"error": "Failed to create request"})
	}

	// IMPORTANT: Set User-Agent as required by Nominatim usage policy
	req.Header.Set("User-Agent", "HVAC-Service/1.0")
	req.Header.Set("Referer", "https://hvac-system.local")

	// Execute Request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ [GEOCODE] Failed to contact Nominatim: %v\n", err)
		return e.JSON(502, map[string]string{"error": "Failed to contact Nominatim"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ [GEOCODE] Nominatim returned status: %d\n", resp.StatusCode)
		return e.JSON(resp.StatusCode, map[string]string{"error": "Nominatim error"})
	}

	// Decode and Proxy Response
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("❌ [GEOCODE] Invalid JSON: %v\n", err)
		return e.JSON(500, map[string]string{"error": "Invalid JSON from Nominatim"})
	}

	return e.JSON(200, result)
}

// -------------------------------------------------------------------
// BRANDING & ASSETS
// -------------------------------------------------------------------

// GetManifest serves dynamic PWA manifest
// GET /manifest.json
func (h *PublicHandler) GetManifest(e *core.RequestEvent) error {
	// 1. Determine Brand (Phase 2: Check URL/Cookie)
	// Phase 1 fallback: Get Default Brand
	var brand *domain.Brand
	var err error

	if h.BrandRepo != nil {
		brand, err = h.BrandRepo.GetDefault()
	}

	// Default fallback values
	name := "HVAC System"
	shortName := "HVAC"
	iconPath := "/assets/images/logo.png" // Default default

	if err == nil && brand != nil {
		if brand.CompanyName != "" {
			name = brand.CompanyName
			shortName = brand.CompanyName // Or truncate?
		}
		if brand.Icon != "" {
			iconPath = fmt.Sprintf("/api/files/brands/%s/%s", brand.Id, brand.Icon)
		} else if brand.Logo != "" {
			iconPath = fmt.Sprintf("/api/files/brands/%s/%s", brand.Id, brand.Logo)
		}
	}

	manifest := map[string]interface{}{
		"name":             name,
		"short_name":       shortName,
		"start_url":        "/",
		"display":          "standalone",
		"background_color": "#ffffff",
		"theme_color":      "#0284c7", // Customize later?
		"icons": []map[string]string{
			{
				"src":   iconPath,
				"sizes": "192x192",
				"type":  "image/png",
			},
			{
				"src":   iconPath,
				"sizes": "512x512",
				"type":  "image/png",
			},
		},
	}

	return e.JSON(200, manifest)
}
