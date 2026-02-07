package services

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type InvoiceService struct {
	app core.App
}

func NewInvoiceService(app core.App) *InvoiceService {
	return &InvoiceService{app: app}
}

// GenerateInvoice creates an invoice based on labor and parts used
func (s *InvoiceService) GenerateInvoice(bookingID string) (*core.Record, error) {
	// Fetch booking
	booking, err := s.app.FindRecordById("bookings", bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking not found")
	}

	// Fetch job report
	jobReports, err := s.app.FindRecordsByFilter(
		"job_reports",
		fmt.Sprintf("booking_id='%s'", bookingID),
		"",
		1,
		0,
		nil,
	)
	if err != nil || len(jobReports) == 0 {
		// Log warning: No report found, but continue with base price?
		// For now, require report
		return nil, fmt.Errorf("job report not found, cannot generate invoice")
	}
	report := jobReports[0]

	// Calculate parts total from job_parts
	partsTotal := 0.0
	jobParts, _ := s.app.FindRecordsByFilter(
		"job_parts",
		fmt.Sprintf("job_report_id='%s'", report.Id),
		"",
		100,
		0,
		nil,
	)
	for _, part := range jobParts {
		partsTotal += part.GetFloat("total")
	}

	// Get base service price (labor)
	serviceID := booking.GetString("service_id")
	laborTotal := 0.0
	serviceName := "Dịch vụ"

	if serviceID == "" {
		fmt.Printf("⚠️  INVOICEGEN: Booking %s has no service_id! Labor will be 0\n", bookingID)
	} else {
		service, err := s.app.FindRecordById("services", serviceID)
		if err != nil {
			fmt.Printf("❌ INVOICEGEN: Service %s not found for booking %s: %v\n", serviceID, bookingID, err)
		} else {
			laborTotal = service.GetFloat("price")
			serviceName = service.GetString("name")
			fmt.Printf("✅ INVOICEGEN: Found service '%s' with price %.2f\n", serviceName, laborTotal)
		}
	}

	totalAmount := partsTotal + laborTotal
	fmt.Printf("DEBUG INVOICEGEN: Booking=%s, ServiceID=%s, Labor=%.2f, Parts=%.2f, Total=%.2f\n",
		bookingID, serviceID, laborTotal, partsTotal, totalAmount)

	// Check for existing invoice
	existingInvoices, _ := s.app.FindRecordsByFilter(
		"invoices",
		fmt.Sprintf("booking_id='%s'", bookingID),
		"",
		1,
		0,
		nil,
	)

	var invoice *core.Record
	invoicesCollection, _ := s.app.FindCollectionByNameOrId("invoices")

	if len(existingInvoices) > 0 {
		invoice = existingInvoices[0]
		// Retain existing status if valid, or reset? Typically retain.
		// Retain public_hash
		fmt.Printf("DEBUG INVOICEGEN: Updating existing invoice %s\n", invoice.Id)
	} else {
		invoice = core.NewRecord(invoicesCollection)
		invoice.Set("booking_id", bookingID)
		invoice.Set("status", "unpaid")
		invoice.Set("public_hash", fmt.Sprintf("%x", time.Now().UnixNano()))
		fmt.Printf("DEBUG INVOICEGEN: Creating new invoice\n")
	}

	invoice.Set("parts_total", partsTotal)
	invoice.Set("labor_total", laborTotal)
	invoice.Set("total_amount", totalAmount)

	// [NEW] Calculate Tech Commission
	// Priority: Tech Rate > Service Rate > Default 10%
	techID := booking.GetString("technician_id")
	techCommission := 0.0
	if techID != "" && laborTotal > 0 {
		commissionRate := 10.0 // Default 10%

		// Check technician's personal rate
		tech, err := s.app.FindRecordById("technicians", techID)
		if err == nil && tech != nil {
			techRate := tech.GetFloat("commission_rate")
			if techRate > 0 {
				commissionRate = techRate
				fmt.Printf("✅ COMMISSION: Using Tech rate %.1f%%\n", commissionRate)
			} else if serviceID != "" {
				// Fallback to service rate
				service, err := s.app.FindRecordById("services", serviceID)
				if err == nil && service != nil {
					serviceRate := service.GetFloat("commission_rate")
					if serviceRate > 0 {
						commissionRate = serviceRate
						fmt.Printf("✅ COMMISSION: Using Service rate %.1f%%\n", commissionRate)
					}
				}
			}
		}

		techCommission = laborTotal * (commissionRate / 100)
		fmt.Printf("✅ COMMISSION: Calculated %.2f (%.1f%% of %.2f labor)\n", techCommission, commissionRate, laborTotal)
	}
	invoice.Set("tech_commission", techCommission)

	if err := s.app.Save(invoice); err != nil {
		return nil, err
	}

	// [NEW] Generate invoice_items for detailed display
	// Delete old items to prevent duplicates when recalculating
	existingItems, _ := s.app.FindRecordsByFilter(
		"invoice_items",
		fmt.Sprintf("invoice_id='%s'", invoice.Id),
		"",
		100,
		0,
		nil,
	)
	for _, item := range existingItems {
		s.app.Delete(item)
	}
	fmt.Printf("DEBUG INVOICEGEN: Deleted %d old invoice items\n", len(existingItems))

	// Create items collection reference
	itemsCollection, err := s.app.FindCollectionByNameOrId("invoice_items")
	if err != nil {
		fmt.Printf("WARNING: invoice_items collection not found: %v\n", err)
		// Continue without items - invoice is still valid
		return invoice, nil
	}

	// Create labor item
	if laborTotal > 0 {
		laborItem := core.NewRecord(itemsCollection)
		laborItem.Set("invoice_id", invoice.Id)
		laborItem.Set("item_name", serviceName)
		laborItem.Set("quantity", 1)
		laborItem.Set("unit_price", laborTotal)
		laborItem.Set("total", laborTotal)
		if err := s.app.Save(laborItem); err != nil {
			fmt.Printf("WARNING: Failed to save labor item: %v\n", err)
		}
	}

	// Create items for each part
	for _, part := range jobParts {
		// Expand item to get name
		s.app.ExpandRecord(part, []string{"item_id"}, nil)
		inventoryItem := part.ExpandedOne("item_id")
		itemName := "Vật tư không tên"
		if inventoryItem != nil {
			itemName = inventoryItem.GetString("name")
		}

		partItem := core.NewRecord(itemsCollection)
		partItem.Set("invoice_id", invoice.Id)
		partItem.Set("item_name", itemName)
		partItem.Set("quantity", part.GetFloat("quantity"))         // Use GetFloat for quantity as it can be decimal
		partItem.Set("unit_price", part.GetFloat("price_per_unit")) // Fix: read price_per_unit
		partItem.Set("total", part.GetFloat("total"))
		if err := s.app.Save(partItem); err != nil {
			fmt.Printf("WARNING: Failed to save part item: %v\n", err)
		}
	}

	fmt.Printf("✅ INVOICEGEN: Created %d invoice items (1 labor + %d parts)\n", 1+len(jobParts), len(jobParts))

	return invoice, nil
}
