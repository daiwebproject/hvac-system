package services

import (
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

// InventoryItem represents a part in stock
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

// JobPart represents a part used in a job
type JobPart struct {
	ItemID       string
	ItemName     string
	Quantity     float64
	PricePerUnit float64
	Total        float64
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
