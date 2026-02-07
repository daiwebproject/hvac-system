package handlers

import (
	"encoding/json" // <-- Nhớ thêm import này
	"fmt"
	"html/template"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/services"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// AdminToolsHandler handles admin management operations
type AdminToolsHandler struct {
	App              core.App
	Templates        *template.Template
	SlotService      *services.TimeSlotService
	InventoryService *services.InventoryService
	Broker           *broker.SegmentedBroker
}

// ShowSlotManager displays the time slot management page
func (h *AdminToolsHandler) ShowSlotManager(e *core.RequestEvent) error {
	// [SMART SCHEDULING] Fetch active technicians for dynamic capacity display
	activeTechs, _ := h.App.FindRecordsByFilter("technicians", "active=true", "name", 100, 0, nil)

	techData := []map[string]interface{}{}
	for _, tech := range activeTechs {
		techData = append(techData, map[string]interface{}{
			"id":     tech.Id,
			"name":   tech.GetString("name"),
			"phone":  tech.GetString("phone"),
			"status": tech.GetString("tech_status"),
		})
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/slots.html", map[string]interface{}{
		"ActiveTechCount": len(activeTechs),
		"ActiveTechs":     techData,
	})
}

// GenerateSlotsForWeek creates slots for the next 7 days
// POST /admin/tools/slots/generate-week
func (h *AdminToolsHandler) GenerateSlotsForWeek(e *core.RequestEvent) error {
	techCountStr := e.Request.FormValue("tech_count")
	techCount, err := strconv.Atoi(techCountStr)
	if err != nil || techCount < 1 {
		// [SMART SCHEDULING] Auto-detect active technicians count
		activeTechs, _ := h.App.FindRecordsByFilter("technicians", "active=true", "", 0, 0, nil)
		techCount = len(activeTechs)
		if techCount == 0 {
			return e.JSON(400, map[string]string{"error": "Không có thợ nào đang trực (active=true)"})
		}
	}

	var errors []string
	var successCount int

	// Generate slots for next 7 days
	for i := 1; i <= 7; i++ {
		date := time.Now().AddDate(0, 0, i).Format("2006-01-02")
		err := h.SlotService.GenerateDefaultSlots(date, techCount)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", date, err.Error()))
		} else {
			successCount++
		}
	}

	result := map[string]interface{}{
		"success_count": successCount,
		"errors":        errors,
		"total_days":    7,
		"tech_count":    techCount,
	}

	return e.JSON(200, result)
}

// ShowInventoryManager hiển thị trang quản lý kho
func (h *AdminToolsHandler) ShowInventoryManager(e *core.RequestEvent) error {
	fmt.Println("🔍 ShowInventoryManager: Starting...")

	// 1. Lấy dữ liệu từ Database (PocketBase)
	items, err := h.InventoryService.GetActiveItems()
	if err != nil {
		fmt.Println("❌ Error loading inventory from service:", err)
		return e.String(500, "Error loading inventory: "+err.Error())
	}
	fmt.Printf("✅ Loaded %d items from DB\n", len(items))

	// 2. Định nghĩa cấu trúc JSON cho Frontend (JavaScript)
	//    Lưu ý: Các `json:"..."` phải khớp chính xác với biến trong file HTML/JS
	type InventoryItemJSON struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		SKU           string  `json:"sku"`
		Category      string  `json:"category"`
		Price         float64 `json:"price"`
		StockQuantity float64 `json:"stock_quantity"` // Quan trọng: Khớp với item.stock_quantity ở JS
		Unit          string  `json:"unit"`
	}

	// 3. Chuyển đổi dữ liệu PocketBase Record -> Struct JSON
	var itemsList []InventoryItemJSON
	for _, item := range items {
		itemsList = append(itemsList, InventoryItemJSON{
			ID:            item.ID,
			Name:          item.Name,
			SKU:           item.SKU,
			Category:      item.Category,
			Price:         item.Price,
			StockQuantity: float64(item.StockQuantity), // Lấy đúng trường từ DB
			Unit:          item.Unit,
		})
	}

	// 4. Mã hóa thành chuỗi JSON
	itemsJSON, err := json.Marshal(itemsList)
	if err != nil {
		fmt.Println("❌ Error marshaling JSON:", err)
		return e.String(500, "JSON Error")
	}

	// Xử lý trường hợp danh sách rỗng để tránh lỗi "null" ở frontend
	if len(itemsList) == 0 {
		itemsJSON = []byte("[]")
	}

	fmt.Println("✅ Rendering page: admin/inventory.html")
	// 5. Gửi xuống View
	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/inventory.html", map[string]interface{}{
		"ItemsJSON": template.JS(string(itemsJSON)), // Biến này sẽ được dùng trong x-data
	})
}

// CreateInventoryItem adds a new part to inventory
// POST /admin/tools/inventory/create
func (h *AdminToolsHandler) CreateInventoryItem(e *core.RequestEvent) error {
	fmt.Println("🔍 CreateInventoryItem: Received Request")

	collection, err := h.App.FindCollectionByNameOrId("inventory_items")
	if err != nil {
		fmt.Println("❌ Collection 'inventory_items' not found:", err)
		return e.JSON(500, map[string]string{"error": "Collection not found"})
	}

	record := core.NewRecord(collection)

	// Parse form values
	name := e.Request.FormValue("name")
	sku := e.Request.FormValue("sku")
	category := e.Request.FormValue("category")
	priceStr := e.Request.FormValue("price")
	stockStr := e.Request.FormValue("stock_quantity")
	unit := e.Request.FormValue("unit")
	description := e.Request.FormValue("description")

	fmt.Printf("📝 Data: Name=%s, SKU=%s, Category=%s, Price=%s, Stock=%s\n", name, sku, category, priceStr, stockStr)

	if name == "" || priceStr == "" {
		return e.JSON(400, map[string]string{"error": "Name and price are required"})
	}

	price, _ := strconv.ParseFloat(priceStr, 64)
	stock, _ := strconv.ParseFloat(stockStr, 64)

	record.Set("name", name)
	record.Set("sku", sku)
	record.Set("category", category)
	record.Set("price", price)
	record.Set("stock_quantity", stock)
	record.Set("unit", unit)
	record.Set("description", description)
	record.Set("is_active", true)

	if err := h.App.Save(record); err != nil {
		fmt.Println("❌ Error saving record:", err)
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	fmt.Println("✅ Inventory Item Created:", record.Id)

	// Construct the item object to return
	newItem := map[string]interface{}{
		"id":             record.Id,
		"name":           name,
		"sku":            sku,
		"category":       category,
		"price":          price,
		"stock_quantity": stock,
		"unit":           unit,
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"item":    newItem,
	})
}

// UpdateInventoryStock updates stock quantity
// POST /admin/tools/inventory/{id}/stock
func (h *AdminToolsHandler) UpdateInventoryStock(e *core.RequestEvent) error {
	itemID := e.Request.PathValue("id")
	quantityStr := e.Request.FormValue("quantity")
	operation := e.Request.FormValue("operation") // "add" or "set"

	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		return e.JSON(400, map[string]string{"error": "Invalid quantity"})
	}

	item, err := h.App.FindRecordById("inventory_items", itemID)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Item not found"})
	}

	currentStock := item.GetFloat("stock_quantity")

	var newStock float64
	if operation == "add" {
		newStock = currentStock + quantity
	} else {
		newStock = quantity
	}

	item.Set("stock_quantity", newStock)

	if err := h.App.Save(item); err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	// Check for low stock alert
	if newStock < 5 { // Threshold = 5
		h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "stock.low",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"item_id":  itemID,
				"name":     item.GetString("name"),
				"quantity": newStock,
				"message":  fmt.Sprintf("Cảnh báo: %s chỉ còn %.0f đơn vị", item.GetString("name"), newStock),
			},
		})
	}

	return e.JSON(200, map[string]interface{}{
		"success":   true,
		"old_stock": currentStock,
		"new_stock": newStock,
	})
}

// ImportToMain adds stock to main warehouse
// POST /admin/tools/inventory/import
func (h *AdminToolsHandler) ImportToMain(e *core.RequestEvent) error {
	productID := e.Request.FormValue("product_id")
	qtyStr := e.Request.FormValue("quantity")
	note := e.Request.FormValue("note")

	if productID == "" || qtyStr == "" {
		return e.JSON(400, map[string]string{"error": "product_id và quantity là bắt buộc"})
	}

	qty, err := strconv.ParseFloat(qtyStr, 64)
	if err != nil || qty <= 0 {
		return e.JSON(400, map[string]string{"error": "Số lượng không hợp lệ"})
	}

	adminID := ""
	if e.Auth != nil {
		adminID = e.Auth.Id
	}

	err = h.InventoryService.ImportToMain(productID, qty, note, adminID)
	if err != nil {
		return e.JSON(400, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Đã nhập %.1f vào kho", qty),
	})
}

// GetLowStockAlerts returns products below threshold
// GET /admin/tools/inventory/alerts
func (h *AdminToolsHandler) GetLowStockAlerts(e *core.RequestEvent) error {
	alerts, err := h.InventoryService.GetLowStockAlerts()
	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// ============================================
// TRUCK STOCK (Kho Trên Xe) - Admin Handlers
// ============================================

// ShowTechStock displays the tech stock management page
// GET /admin/tools/tech-stock
func (h *AdminToolsHandler) ShowTechStock(e *core.RequestEvent) error {
	fmt.Println("🔍 ShowTechStock: Loading...")

	// Get all active technicians
	techs, err := h.App.FindRecordsByFilter("technicians", "active=true", "name", 100, 0, nil)
	if err != nil {
		return e.String(500, "Error loading technicians: "+err.Error())
	}

	// Build technician list with stock summary
	type TechWithStock struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Phone      string  `json:"phone"`
		ItemCount  int     `json:"item_count"`
		TotalValue float64 `json:"total_value"`
	}

	var techList []TechWithStock
	for _, tech := range techs {
		techID := tech.Id
		items, _ := h.InventoryService.GetTechInventory(techID)

		totalValue := 0.0
		for _, item := range items {
			totalValue += item.Price * item.Quantity
		}

		techList = append(techList, TechWithStock{
			ID:         techID,
			Name:       tech.GetString("name"),
			Phone:      tech.GetString("phone"),
			ItemCount:  len(items),
			TotalValue: totalValue,
		})
	}

	// Get all inventory items for transfer dropdown
	allItems, _ := h.InventoryService.GetActiveItems()

	// Map to JSON struct with lowercase tags
	type InventoryItemJSON struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		SKU           string  `json:"sku"`
		Category      string  `json:"category"`
		Price         float64 `json:"price"`
		StockQuantity float64 `json:"stock_quantity"`
		Unit          string  `json:"unit"`
	}

	var itemsList []InventoryItemJSON
	for _, item := range allItems {
		itemsList = append(itemsList, InventoryItemJSON{
			ID:            item.ID,
			Name:          item.Name,
			SKU:           item.SKU,
			Category:      item.Category,
			Price:         item.Price,
			StockQuantity: float64(item.StockQuantity),
			Unit:          item.Unit,
		})
	}

	techsJSON, _ := json.Marshal(techList)
	itemsJSON, _ := json.Marshal(itemsList)

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/tech_stock.html", map[string]interface{}{
		"TechsJSON": template.JS(string(techsJSON)),
		"ItemsJSON": template.JS(string(itemsJSON)),
	})
}

// ShowTechStockDetail displays inventory for a specific technician
// GET /admin/tools/tech-stock/{id}
func (h *AdminToolsHandler) ShowTechStockDetail(e *core.RequestEvent) error {
	techID := e.Request.PathValue("id")

	tech, err := h.App.FindRecordById("technicians", techID)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Technician not found"})
	}

	items, err := h.InventoryService.GetTechInventory(techID)
	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]interface{}{
		"technician": map[string]string{
			"id":    tech.Id,
			"name":  tech.GetString("name"),
			"phone": tech.GetString("phone"),
		},
		"items": items,
	})
}

// TransferStock transfers items from main inventory to technician
// POST /admin/tools/tech-stock/transfer
func (h *AdminToolsHandler) TransferStock(e *core.RequestEvent) error {
	techID := e.Request.FormValue("technician_id")
	itemID := e.Request.FormValue("item_id")
	qtyStr := e.Request.FormValue("quantity")
	adminID := "admin" // TODO: Get from session

	fmt.Printf("📦 TransferStock: Tech=%s Item=%s Qty=%s\n", techID, itemID, qtyStr)

	if techID == "" || itemID == "" || qtyStr == "" {
		fmt.Println("❌ Missing fields in TransferStock")
		return e.JSON(400, map[string]string{"error": "Missing required fields"})
	}

	qty, err := strconv.ParseFloat(qtyStr, 64)
	if err != nil || qty <= 0 {
		fmt.Println("❌ Invalid quantity in TransferStock:", qtyStr)
		return e.JSON(400, map[string]string{"error": "Invalid quantity"})
	}

	err = h.InventoryService.TransferToTech(itemID, techID, qty, adminID)
	if err != nil {
		fmt.Println("❌ TransferToTech Error:", err)
		return e.JSON(400, map[string]string{"error": err.Error()})
	}

	// Get item name for SSE notification
	item, _ := h.App.FindRecordById("inventory_items", itemID)
	itemName := ""
	if item != nil {
		itemName = item.GetString("name")
	}

	// Notify via SSE
	h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
		Type:      "stock.transfer",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"type":     "main_to_tech",
			"tech_id":  techID,
			"item":     itemName,
			"quantity": qty,
			"message":  fmt.Sprintf("Đã cấp %.1f %s cho thợ", qty, itemName),
		},
	})

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Đã cấp %.1f %s", qty, itemName),
	})
}

// ReturnStock returns items from technician back to main inventory
// POST /admin/tools/tech-stock/return
func (h *AdminToolsHandler) ReturnStock(e *core.RequestEvent) error {
	techID := e.Request.FormValue("technician_id")
	itemID := e.Request.FormValue("item_id")
	qtyStr := e.Request.FormValue("quantity")
	adminID := "admin"

	if techID == "" || itemID == "" || qtyStr == "" {
		return e.JSON(400, map[string]string{"error": "Missing required fields"})
	}

	qty, err := strconv.ParseFloat(qtyStr, 64)
	if err != nil || qty <= 0 {
		return e.JSON(400, map[string]string{"error": "Invalid quantity"})
	}

	err = h.InventoryService.ReturnToMain(techID, itemID, qty, adminID)
	if err != nil {
		return e.JSON(400, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Đã trả hàng về kho chính",
	})
}

// UpdateInventoryItem updates an existing inventory item
// POST /admin/tools/inventory/{id}/update
func (h *AdminToolsHandler) UpdateInventoryItem(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	fmt.Printf("🔍 UpdateInventoryItem: Updating item %s\n", id)

	record, err := h.App.FindRecordById("inventory_items", id)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Item not found"})
	}

	// Parse form values
	name := e.Request.FormValue("name")
	sku := e.Request.FormValue("sku")
	category := e.Request.FormValue("category")
	priceStr := e.Request.FormValue("price")
	unit := e.Request.FormValue("unit")
	description := e.Request.FormValue("description")
	// stock_quantity is usually managed via stock updates but can be set here if needed
	// For now let's allow updating everything EXCEPT stock if intended, or just everything.
	// User request was "edit information", usually stock is separate, but "Manage Item" modal has stock field.
	// Let's allow stock update too if provided, but typically stock operations should be logged.
	// For simplicity in "Edit Info", we might accept it if the user changes it.
	stockStr := e.Request.FormValue("stock_quantity")

	if name == "" || priceStr == "" {
		return e.JSON(400, map[string]string{"error": "Name and price are required"})
	}

	price, _ := strconv.ParseFloat(priceStr, 64)
	stock, _ := strconv.ParseFloat(stockStr, 64)

	record.Set("name", name)
	record.Set("sku", sku)
	record.Set("category", category)
	record.Set("price", price)
	record.Set("stock_quantity", stock)
	record.Set("unit", unit)
	record.Set("description", description)

	if err := h.App.Save(record); err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	newItem := map[string]interface{}{
		"id":             record.Id,
		"name":           name,
		"sku":            sku,
		"category":       category,
		"price":          price,
		"stock_quantity": stock,
		"unit":           unit,
		"description":    description,
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Cập nhật thành công",
		"item":    newItem,
	})
}

// DeleteInventoryItem soft deletes an inventory item
// POST /admin/tools/inventory/{id}/delete
func (h *AdminToolsHandler) DeleteInventoryItem(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	fmt.Printf("🔍 DeleteInventoryItem: Deleting item %s\n", id)

	record, err := h.App.FindRecordById("inventory_items", id)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Item not found"})
	}

	// Soft delete
	record.Set("is_active", false)

	if err := h.App.Save(record); err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Đã xóa vật tư",
		"id":      id,
	})
}
