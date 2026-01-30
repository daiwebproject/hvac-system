package services

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type BookingService struct {
	App core.App
}

func NewBookingService(app core.App) *BookingService {
	return &BookingService{App: app}
}

type BookingRequest struct {
	ServiceID    string
	CustomerName string
	Phone        string
	Address      string
	IssueDesc    string
	DeviceType   string
	Brand        string
	BookingTime  string // \Deprecated: Use SlotID instead
	SlotID       string // Time slot reservation
	Lat          float64
	Long         float64
	Files        []*filesystem.File
}

// CreateBooking creates a new booking record
// If SlotID is provided, it will also reserve the time slot
func (s *BookingService) CreateBooking(req BookingRequest) (*core.Record, error) {
	collection, err := s.App.FindCollectionByNameOrId("bookings")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)

	if req.ServiceID != "check" {
		record.Set("service_id", req.ServiceID)
	}
	record.Set("customer_name", req.CustomerName)
	record.Set("customer_phone", req.Phone)
	record.Set("address_details", req.Address)
	record.Set("issue_description", req.IssueDesc)

	// Backward compatibility: support both SlotID and BookingTime
	if req.SlotID != "" {
		record.Set("time_slot_id", req.SlotID)
		// TODO: Integrate TimeSlotService to actually reserve the slot
		// This will be handled by the handler layer to avoid circular dependency
	} else if req.BookingTime != "" {
		record.Set("booking_time", req.BookingTime)
	}

	// Location
	if req.Lat != 0 {
		record.Set("lat", req.Lat)
		record.Set("long", req.Long)
	}
	record.Set("device_type", req.DeviceType)
	record.Set("brand", req.Brand)
	record.Set("job_status", "pending")

	// Handling files
	if len(req.Files) > 0 {
		// Convert []*filesystem.File to []any
		fileSlice := make([]any, len(req.Files))
		for i, f := range req.Files {
			fileSlice[i] = f
		}
		record.Set("client_images", fileSlice)
	}

	if err := s.App.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}
