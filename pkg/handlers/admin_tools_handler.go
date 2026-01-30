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
	// Use layout inheritance
	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/slots.html", nil)
}

// GenerateSlotsForWeek creates slots for the next 7 days
// POST /admin/tools/slots/generate-week
func (h *AdminToolsHandler) GenerateSlotsForWeek(e *core.RequestEvent) error {
	techCountStr := e.Request.FormValue("tech_count")
	techCount, err := strconv.Atoi(techCountStr)
	if err != nil || techCount < 1 {
		techCount = 2 // Default to 2 technicians
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
	}

	return e.JSON(200, result)
}

// ShowInventoryManager hiển thị trang quản lý kho
func (h *AdminToolsHandler) ShowInventoryManager(e *core.RequestEvent) error {
	// 1. Lấy dữ liệu từ Database (PocketBase)
	items, err := h.InventoryService.GetActiveItems()
	if err != nil {
		return e.String(500, "Error loading inventory")
	}
	fmt.Println(items)

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
	itemsJSON, _ := json.Marshal(itemsList)

	// Xử lý trường hợp danh sách rỗng để tránh lỗi "null" ở frontend
	if len(itemsList) == 0 {
		itemsJSON = []byte("[]")
	}

	// 5. Gửi xuống View
	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/inventory.html", map[string]interface{}{
		"ItemsJSON": template.JS(string(itemsJSON)), // Biến này sẽ được dùng trong x-data
	})
}

// CreateInventoryItem adds a new part to inventory
// POST /admin/tools/inventory/create
func (h *AdminToolsHandler) CreateInventoryItem(e *core.RequestEvent) error {
	collection, err := h.App.FindCollectionByNameOrId("inventory_items")
	if err != nil {
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
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"item_id": record.Id,
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
