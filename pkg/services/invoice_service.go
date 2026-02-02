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
	service, err := s.app.FindRecordById("services", serviceID)
	laborTotal := 0.0
	if err == nil {
		laborTotal = service.GetFloat("price")
	}

	totalAmount := partsTotal + laborTotal

	// Create Invoice
	invoices, _ := s.app.FindCollectionByNameOrId("invoices")
	invoice := core.NewRecord(invoices)
	invoice.Set("booking_id", bookingID)
	invoice.Set("parts_total", partsTotal)
	invoice.Set("labor_total", laborTotal)
	invoice.Set("total_amount", totalAmount)
	invoice.Set("status", "unpaid")
	invoice.Set("public_hash", fmt.Sprintf("%x", time.Now().UnixNano()))

	if err := s.app.Save(invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}
