package repository

import (
	"hvac-system/internal/core"

	"github.com/pocketbase/dbx"
	pbCore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type PBBookingRepo struct {
	app pbCore.App
}

func NewBookingRepo(app pbCore.App) core.BookingRepository {
	return &PBBookingRepo{app: app}
}

// Mapping helper: Record -> Domain Model
func (r *PBBookingRepo) toDomain(record *pbCore.Record) *core.Booking {
	slotID := record.GetString("slot_id")
	var slotIDPtr *string
	if slotID != "" {
		slotIDPtr = &slotID
	}

	return &core.Booking{
		ID:               record.Id,
		ServiceID:        record.GetString("service_id"),
		CustomerName:     record.GetString("customer_name"),
		CustomerPhone:    record.GetString("customer_phone"),
		Address:          record.GetString("address"),
		AddressDetails:   record.GetString("address_details"),
		IssueDescription: record.GetString("issue_description"),
		DeviceType:       record.GetString("device_type"),
		Brand:            record.GetString("brand"),
		JobStatus:        record.GetString("job_status"),
		BookingTime:      record.GetString("booking_time"),
		TechnicianID:     record.GetString("technician_id"),
		SlotID:           slotIDPtr,
		Created:          record.GetString("created"),
		Updated:          record.GetString("updated"),
		Lat:              record.GetFloat("lat"),
		Long:             record.GetFloat("long"),
	}
}

// GetByID fetches a booking by ID
func (r *PBBookingRepo) GetByID(id string) (*core.Booking, error) {
	record, err := r.app.FindRecordById("bookings", id)
	if err != nil {
		return nil, err
	}
	return r.toDomain(record), nil
}

// Create persists a new booking
func (r *PBBookingRepo) Create(b *core.Booking, files []*filesystem.File) error {
	collection, err := r.app.FindCollectionByNameOrId("bookings")
	if err != nil {
		return err
	}

	record := pbCore.NewRecord(collection)

	// Map Domain -> Record
	record.Set("service_id", b.ServiceID)
	record.Set("customer_name", b.CustomerName)
	record.Set("customer_phone", b.CustomerPhone)
	record.Set("address_details", b.AddressDetails) // mapped from b.AddressDetails? Check usage.
	record.Set("address", b.Address)                // mapped from b.Address
	record.Set("issue_description", b.IssueDescription)
	record.Set("device_type", b.DeviceType)
	record.Set("brand", b.Brand)
	record.Set("job_status", b.JobStatus)

	if b.BookingTime != "" {
		record.Set("booking_time", b.BookingTime)
	}

	if b.SlotID != nil {
		record.Set("time_slot_id", *b.SlotID)
	}

	if b.Lat != 0 {
		record.Set("lat", b.Lat)
		record.Set("long", b.Long)
	}

	// Handle files
	if len(files) > 0 {
		fileSlice := make([]any, len(files))
		for i, f := range files {
			fileSlice[i] = f
		}
		record.Set("client_images", fileSlice)
	}

	if err := r.app.Save(record); err != nil {
		return err
	}

	// Update ID and Timestamps back to domain model
	b.ID = record.Id
	b.Created = record.GetString("created")
	b.Updated = record.GetString("updated")

	return nil
}

// Update persists changes to an existing booking
func (r *PBBookingRepo) Update(b *core.Booking) error {
	record, err := r.app.FindRecordById("bookings", b.ID)
	if err != nil {
		return err
	}

	// Update fields
	record.Set("technician_id", b.TechnicianID)
	record.Set("job_status", b.JobStatus)

	if b.SlotID != nil {
		record.Set("slot_id", *b.SlotID)
	} else {
		record.Set("slot_id", nil)
	}

	// Fields that might change during update
	// Note: Be careful not to overwrite with empty values if partial update is intended
	// Logic here assumes 'b' is a complete object.

	// ... (Map other fields as necessary, for now we focus on critical ones for State changes)

	// Reset specific fields if needed (like technician removal)
	if b.TechnicianID == "" {
		record.Set("technician_id", nil)
		// Assuming we also reset related timestamps as per recall logic
		record.Set("moving_start_at", nil)
		record.Set("working_start_at", nil)
		record.Set("completed_at", nil)
	}

	if err := r.app.Save(record); err != nil {
		return err
	}

	return nil
}

func (r *PBBookingRepo) FindPending() ([]*core.Booking, error) {
	records, err := r.app.FindRecordsByFilter("bookings", "job_status = 'pending'", "-created", 0, 0, nil)
	if err != nil {
		return nil, err
	}

	var bookings []*core.Booking
	for _, rec := range records {
		bookings = append(bookings, r.toDomain(rec))
	}
	return bookings, nil
}

func (r *PBBookingRepo) FindActiveByTechnician(techID string) ([]*core.Booking, error) {
	// Active = Not Completed and Not Cancelled
	records, err := r.app.FindRecordsByFilter(
		"bookings",
		"technician_id = {:techId} && job_status != 'completed' && job_status != 'cancelled'",
		"",
		0, 0,
		dbx.Params{"techId": techID},
	)

	if err != nil {
		return nil, err
	}

	var bookings []*core.Booking
	for _, rec := range records {
		bookings = append(bookings, r.toDomain(rec))
	}
	return bookings, nil
}

func (r *PBBookingRepo) FindScheduledByTechnician(techID string) ([]*core.Booking, error) {
	// Scheduled = Not Cancelled
	records, err := r.app.FindRecordsByFilter(
		"bookings",
		"technician_id = {:techId} && job_status != 'cancelled'",
		"",
		0, 0,
		dbx.Params{"techId": techID},
	)

	if err != nil {
		return nil, err
	}

	var bookings []*core.Booking
	for _, rec := range records {
		bookings = append(bookings, r.toDomain(rec))
	}
	return bookings, nil
}
