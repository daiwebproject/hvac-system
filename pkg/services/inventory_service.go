// inventory_service.go
package services

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// InventoryService handles parts inventory and usage tracking
type InventoryService struct {
	app core.App
}

// NewInventoryService creates a new inventory service
func NewInventoryService(app core.App) *InventoryService {
	return &InventoryService{app: app}
}

// DeductStock reduces the stock of an item by quantity and returns the cost
// This is a helper for atomic operations within a transaction context ideally
func (s *InventoryService) DeductStock(itemID string, quantity float64) (float64, error) {
	item, err := s.app.FindRecordById("inventory_items", itemID)
	if err != nil {
		return 0, err
	}

	currentStock := item.GetFloat("stock_quantity")
	if currentStock < quantity {
		return 0, fmt.Errorf("insufficient stock: have %.0f, need %.0f", currentStock, quantity)
	}

	// Deduct
	item.Set("stock_quantity", currentStock-quantity)
	if err := s.app.Save(item); err != nil {
		return 0, err
	}

	return item.GetFloat("price"), nil
}

// InventoryItem represents a part in stock (legacy, for backward compatibility)
type InventoryItem struct {
	ID            string
	Name          string
	SKU           string
	Category      string
	Price         float64
	StockQuantity int
	Unit          string
	IsActive      bool
}

// Product represents a product with pricing info (for multi-location inventory)
type Product struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	SKU          string  `json:"sku"`
	Category     string  `json:"category"`
	Unit         string  `json:"unit"`
	PriceSell    float64 `json:"price_sell"`    // Giá bán
	PriceImport  float64 `json:"price_import"`  // Giá vốn
	MinThreshold float64 `json:"min_threshold"` // Ngưỡng cảnh báo
	MainStock    float64 `json:"main_stock"`    // Tồn kho tổng
	IsActive     bool    `json:"is_active"`
}

// LowStockAlert represents a product below threshold
type LowStockAlert struct {
	Product   Product `json:"product"`
	MainStock float64 `json:"main_stock"`
	Deficit   float64 `json:"deficit"` // How much below threshold
}

// JobPart represents a part used in a job
type JobPart struct {
	ItemID       string  `json:"id"`
	ItemName     string  `json:"name"`
	Quantity     float64 `json:"qty"`
	PricePerUnit float64 `json:"price"`
	Total        float64 `json:"total"`
}

// GetActiveItems returns all active inventory items
func (s *InventoryService) GetActiveItems() ([]InventoryItem, error) {
	records, err := s.app.FindRecordsByFilter(
		"inventory_items",
		"is_active = true",
		"category, name",
		500,
		0,
		nil,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "sql: no rows in result set" {
			return []InventoryItem{}, nil
		}
		return nil, err
	}

	items := make([]InventoryItem, len(records))
	for i, record := range records {
		items[i] = InventoryItem{
			ID:            record.Id,
			Name:          record.GetString("name"),
			SKU:           record.GetString("sku"),
			Category:      record.GetString("category"),
			Price:         record.GetFloat("price"),
			StockQuantity: int(record.GetFloat("stock_quantity")),
			Unit:          record.GetString("unit"),
			IsActive:      record.GetBool("is_active"),
		}
	}

	return items, nil
}

// GetItemsByCategory returns items filtered by category
func (s *InventoryService) GetItemsByCategory(category string) ([]InventoryItem, error) {
	filter := fmt.Sprintf("is_active = true && category = '%s'", category)
	records, err := s.app.FindRecordsByFilter(
		"inventory_items",
		filter,
		"name",
		100,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}

	items := make([]InventoryItem, len(records))
	for i, record := range records {
		items[i] = InventoryItem{
			ID:            record.Id,
			Name:          record.GetString("name"),
			SKU:           record.GetString("sku"),
			Category:      category,
			Price:         record.GetFloat("price"),
			StockQuantity: int(record.GetFloat("stock_quantity")),
			Unit:          record.GetString("unit"),
			IsActive:      true,
		}
	}

	return items, nil
}

// RecordPartsUsage saves parts used in a job and updates stock
// Business rule: Atomic stock deduction to prevent overselling
func (s *InventoryService) RecordPartsUsage(jobReportID string, parts []JobPart) (float64, error) {
	if len(parts) == 0 {
		return 0, nil
	}

	collection, err := s.app.FindCollectionByNameOrId("job_parts")
	if err != nil {
		return 0, err
	}

	var totalPartsCost float64

	for _, part := range parts {
		// Verify item exists and has stock
		item, err := s.app.FindRecordById("inventory_items", part.ItemID)
		if err != nil {
			return 0, fmt.Errorf("item %s not found", part.ItemID)
		}

		currentStock := item.GetFloat("stock_quantity")
		if currentStock < part.Quantity {
			return 0, fmt.Errorf("insufficient stock for %s: need %.0f, have %.0f",
				item.GetString("name"), part.Quantity, currentStock)
		}

		// Get current price (for frozen pricing)
		pricePerUnit := item.GetFloat("price")
		total := part.Quantity * pricePerUnit

		// Create job_parts record
		record := core.NewRecord(collection)
		record.Set("job_report_id", jobReportID)
		record.Set("item_id", part.ItemID)
		record.Set("quantity", part.Quantity)
		record.Set("price_per_unit", pricePerUnit)
		record.Set("total", total)

		if err := s.app.Save(record); err != nil {
			return 0, fmt.Errorf("failed to record part usage: %w", err)
		}

		// Deduct stock
		item.Set("stock_quantity", currentStock-part.Quantity)
		if err := s.app.Save(item); err != nil {
			return 0, fmt.Errorf("failed to update stock: %w", err)
		}

		totalPartsCost += total
	}

	return totalPartsCost, nil
}

// RecordPartsUsageFromTech saves parts used and deducts from TECHNICIAN'S inventory (Truck Stock)
// This is the new workflow for Kho Trên Xe feature
func (s *InventoryService) RecordPartsUsageFromTech(jobReportID, techID, jobID string, parts []JobPart) (float64, error) {
	if len(parts) == 0 {
		return 0, nil
	}

	collection, err := s.app.FindCollectionByNameOrId("job_parts")
	if err != nil {
		return 0, err
	}

	var totalPartsCost float64

	for _, part := range parts {
		// Deduct from tech inventory (will error if insufficient stock)
		pricePerUnit, err := s.DeductTechStock(techID, part.ItemID, part.Quantity, jobID)
		if err != nil {
			return 0, err
		}

		total := part.Quantity * pricePerUnit

		// Create job_parts record
		record := core.NewRecord(collection)
		record.Set("job_report_id", jobReportID)
		record.Set("item_id", part.ItemID)
		record.Set("quantity", part.Quantity)
		record.Set("price_per_unit", pricePerUnit)
		record.Set("total", total)

		if err := s.app.Save(record); err != nil {
			return 0, fmt.Errorf("failed to record part usage: %w", err)
		}

		totalPartsCost += total
	}

	return totalPartsCost, nil
}

// CalculateJobCost calculates total cost including labor and parts
func (s *InventoryService) CalculateJobCost(jobReportID string, laborCost float64) (float64, error) {
	// Fetch all parts used in this job
	filter := fmt.Sprintf("job_report_id = '%s'", jobReportID)
	records, err := s.app.FindRecordsByFilter("job_parts", filter, "", 100, 0, nil)
	if err != nil {
		return 0, err
	}

	var partsCost float64
	for _, record := range records {
		partsCost += record.GetFloat("total")
	}

	return laborCost + partsCost, nil
}

// GetJobParts returns all parts used in a specific job
func (s *InventoryService) GetJobParts(jobReportID string) ([]JobPart, error) {
	filter := fmt.Sprintf("job_report_id = '%s'", jobReportID)
	records, err := s.app.FindRecordsByFilter("job_parts", filter, "", 100, 0, nil)
	if err != nil {
		return nil, err
	}

	// Expand item relation to get names
	for _, record := range records {
		s.app.ExpandRecord(record, []string{"item_id"}, nil)
	}

	parts := make([]JobPart, len(records))
	for i, record := range records {
		item := record.ExpandedOne("item_id")
		itemName := "Unknown"
		if item != nil {
			itemName = item.GetString("name")
		}

		parts[i] = JobPart{
			ItemID:       record.GetString("item_id"),
			ItemName:     itemName,
			Quantity:     record.GetFloat("quantity"),
			PricePerUnit: record.GetFloat("price_per_unit"),
			Total:        record.GetFloat("total"),
		}
	}

	return parts, nil
}

// ============================================
// TRUCK STOCK (Kho Trên Xe) - New Methods
// ============================================

// TechStockItem represents an item in a technician's truck inventory
type TechStockItem struct {
	ID        string // tech_inventory record ID
	TechID    string
	ItemID    string
	ItemName  string
	ItemSKU   string
	Price     float64
	Quantity  float64
	Unit      string
	MainStock float64 // Reference to main inventory stock
}

// GetTechInventory returns all items in a technician's truck stock
func (s *InventoryService) GetTechInventory(techID string) ([]TechStockItem, error) {
	filter := fmt.Sprintf("technician_id = '%s' && quantity > 0", techID)
	records, err := s.app.FindRecordsByFilter(
		"tech_inventory",
		filter,
		"",
		500,
		0,
		nil,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "sql: no rows in result set" {
			return []TechStockItem{}, nil
		}
		return nil, err
	}

	// Expand item relation
	for _, record := range records {
		s.app.ExpandRecord(record, []string{"item_id"}, nil)
	}

	items := make([]TechStockItem, len(records))
	for i, record := range records {
		item := record.ExpandedOne("item_id")
		itemName := ""
		itemSKU := ""
		price := 0.0
		unit := ""
		mainStock := 0.0

		if item != nil {
			itemName = item.GetString("name")
			itemSKU = item.GetString("sku")
			price = item.GetFloat("price")
			unit = item.GetString("unit")
			mainStock = item.GetFloat("stock_quantity")
		}

		items[i] = TechStockItem{
			ID:        record.Id,
			TechID:    techID,
			ItemID:    record.GetString("item_id"),
			ItemName:  itemName,
			ItemSKU:   itemSKU,
			Price:     price,
			Quantity:  record.GetFloat("quantity"),
			Unit:      unit,
			MainStock: mainStock,
		}
	}

	return items, nil
}

// TransferToTech transfers stock from main inventory to a technician's truck
func (s *InventoryService) TransferToTech(itemID, techID string, qty float64, adminID string) error {
	// 1. Check main stock
	item, err := s.app.FindRecordById("inventory_items", itemID)
	if err != nil {
		return fmt.Errorf("vật tư không tồn tại: %w", err)
	}

	mainStock := item.GetFloat("stock_quantity")
	if mainStock < qty {
		return fmt.Errorf("kho chính không đủ hàng: cần %.1f, còn %.1f", qty, mainStock)
	}

	// 2. Deduct from main stock
	item.Set("stock_quantity", mainStock-qty)
	if err := s.app.Save(item); err != nil {
		return fmt.Errorf("lỗi trừ kho chính: %w", err)
	}

	// 3. Add to tech inventory (upsert)
	filter := fmt.Sprintf("technician_id = '%s' && item_id = '%s'", techID, itemID)
	existing, _ := s.app.FindFirstRecordByFilter("tech_inventory", filter)

	if existing != nil {
		// Update existing record
		newQty := existing.GetFloat("quantity") + qty
		existing.Set("quantity", newQty)
		if err := s.app.Save(existing); err != nil {
			return fmt.Errorf("lỗi cập nhật kho thợ: %w", err)
		}
	} else {
		// Create new record
		collection, err := s.app.FindCollectionByNameOrId("tech_inventory")
		if err != nil {
			return err
		}
		record := core.NewRecord(collection)
		record.Set("technician_id", techID)
		record.Set("item_id", itemID)
		record.Set("quantity", qty)
		if err := s.app.Save(record); err != nil {
			return fmt.Errorf("lỗi tạo kho thợ: %w", err)
		}
	}

	// 4. Log transfer
	s.logTransfer("main_to_tech", "", techID, itemID, qty, adminID, "")

	return nil
}

// DeductTechStock deducts from technician's truck stock when completing a job
func (s *InventoryService) DeductTechStock(techID, itemID string, qty float64, jobID string) (float64, error) {
	// Find tech inventory record
	filter := fmt.Sprintf("technician_id = '%s' && item_id = '%s'", techID, itemID)
	record, err := s.app.FindFirstRecordByFilter("tech_inventory", filter)
	if err != nil {
		return 0, fmt.Errorf("vật tư không có trong kho xe")
	}

	currentQty := record.GetFloat("quantity")
	if currentQty < qty {
		return 0, fmt.Errorf("kho xe không đủ: cần %.1f, còn %.1f", qty, currentQty)
	}

	// Deduct
	record.Set("quantity", currentQty-qty)
	if err := s.app.Save(record); err != nil {
		return 0, err
	}

	// Get price for return
	item, _ := s.app.FindRecordById("inventory_items", itemID)
	price := 0.0
	if item != nil {
		price = item.GetFloat("price")
	}

	// Log transfer
	s.logTransfer("tech_to_job", techID, "", itemID, qty, "", jobID)

	return price, nil
}

// ReturnToMain returns stock from technician's truck back to main inventory
func (s *InventoryService) ReturnToMain(techID, itemID string, qty float64, adminID string) error {
	// 1. Check tech stock
	filter := fmt.Sprintf("technician_id = '%s' && item_id = '%s'", techID, itemID)
	techRecord, err := s.app.FindFirstRecordByFilter("tech_inventory", filter)
	if err != nil {
		return fmt.Errorf("vật tư không có trong kho xe")
	}

	currentQty := techRecord.GetFloat("quantity")
	if currentQty < qty {
		return fmt.Errorf("kho xe không đủ để trả: có %.1f, muốn trả %.1f", currentQty, qty)
	}

	// 2. Deduct from tech
	techRecord.Set("quantity", currentQty-qty)
	if err := s.app.Save(techRecord); err != nil {
		return err
	}

	// 3. Add back to main stock
	item, err := s.app.FindRecordById("inventory_items", itemID)
	if err != nil {
		return err
	}
	item.Set("stock_quantity", item.GetFloat("stock_quantity")+qty)
	if err := s.app.Save(item); err != nil {
		return err
	}

	// 4. Log transfer
	s.logTransfer("tech_to_main", techID, "", itemID, qty, adminID, "")

	return nil
}

// logTransfer creates an audit log entry for stock movements
func (s *InventoryService) logTransfer(transferType, fromID, toID, itemID string, qty float64, createdBy, jobID string) {
	collection, err := s.app.FindCollectionByNameOrId("stock_transfers")
	if err != nil {
		return // Silently fail logging - don't block main operation
	}

	record := core.NewRecord(collection)
	record.Set("transfer_type", transferType)
	record.Set("from_id", fromID)
	record.Set("to_id", toID)
	record.Set("item_id", itemID)
	record.Set("quantity", qty)
	record.Set("created_by", createdBy)
	record.Set("job_id", jobID)

	s.app.Save(record) // Best effort
}

// ============================================
// MULTI-LOCATION INVENTORY METHODS
// ============================================

// GetProducts returns all products with main warehouse stock
func (s *InventoryService) GetProducts() ([]Product, error) {
	records, err := s.app.FindRecordsByFilter(
		"inventory_items",
		"is_active = true",
		"category, name",
		500, 0, nil,
	)
	if err != nil {
		return nil, err
	}

	var products []Product
	for _, r := range records {
		p := Product{
			ID:           r.Id,
			Name:         r.GetString("name"),
			SKU:          r.GetString("sku"),
			Category:     r.GetString("category"),
			Unit:         r.GetString("unit"),
			PriceSell:    r.GetFloat("price"),
			PriceImport:  r.GetFloat("price_import"),
			MinThreshold: r.GetFloat("min_threshold"),
			MainStock:    r.GetFloat("stock_quantity"), // Main warehouse stock
			IsActive:     r.GetBool("is_active"),
		}
		products = append(products, p)
	}

	return products, nil
}

// GetMainStock returns the main warehouse stock for a product
func (s *InventoryService) GetMainStock(productID string) (float64, error) {
	item, err := s.app.FindRecordById("inventory_items", productID)
	if err != nil {
		return 0, err
	}
	return item.GetFloat("stock_quantity"), nil
}

// ImportToMain adds stock to main warehouse and logs it
func (s *InventoryService) ImportToMain(productID string, qty float64, note, adminID string) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}

	item, err := s.app.FindRecordById("inventory_items", productID)
	if err != nil {
		return fmt.Errorf("product not found: %w", err)
	}

	currentStock := item.GetFloat("stock_quantity")
	item.Set("stock_quantity", currentStock+qty)

	if err := s.app.Save(item); err != nil {
		return err
	}

	// Log the import
	s.logTransfer("import", "", "main", productID, qty, adminID, "")

	return nil
}

// GetLowStockAlerts returns products below their min_threshold
func (s *InventoryService) GetLowStockAlerts() ([]LowStockAlert, error) {
	// Find products where stock < min_threshold and min_threshold > 0
	records, err := s.app.FindRecordsByFilter(
		"inventory_items",
		"is_active = true && min_threshold > 0",
		"-min_threshold",
		100, 0, nil,
	)
	if err != nil {
		return nil, err
	}

	var alerts []LowStockAlert
	for _, r := range records {
		stock := r.GetFloat("stock_quantity")
		threshold := r.GetFloat("min_threshold")

		if stock < threshold {
			product := Product{
				ID:           r.Id,
				Name:         r.GetString("name"),
				SKU:          r.GetString("sku"),
				Category:     r.GetString("category"),
				Unit:         r.GetString("unit"),
				PriceSell:    r.GetFloat("price"),
				MinThreshold: threshold,
				MainStock:    stock,
			}

			alerts = append(alerts, LowStockAlert{
				Product:   product,
				MainStock: stock,
				Deficit:   threshold - stock,
			})
		}
	}

	return alerts, nil
}
