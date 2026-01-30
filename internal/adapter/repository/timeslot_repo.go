package repository

import (
	"hvac-system/internal/core"

	pbCore "github.com/pocketbase/pocketbase/core"
)

type PBTimeSlotRepo struct {
	app pbCore.App
}

func NewTimeSlotRepo(app pbCore.App) core.TimeSlotRepository {
	return &PBTimeSlotRepo{app: app}
}

func (r *PBTimeSlotRepo) toDomain(record *pbCore.Record) *core.TimeSlot {
	return &core.TimeSlot{
		ID:              record.Id,
		Date:            record.GetString("date"),
		StartTime:       record.GetString("start_time"),
		EndTime:         record.GetString("end_time"),
		MaxCapacity:     record.GetInt("max_capacity"),
		CurrentBookings: record.GetInt("current_bookings"),
		IsBooked:        record.GetBool("is_booked"),
		IsAvailable:     record.GetInt("current_bookings") < record.GetInt("max_capacity"),
	}
}

func (r *PBTimeSlotRepo) GetByID(id string) (*core.TimeSlot, error) {
	record, err := r.app.FindRecordById("time_slots", id)
	if err != nil {
		return nil, err
	}
	return r.toDomain(record), nil
}

func (r *PBTimeSlotRepo) Update(slot *core.TimeSlot) error {
	record, err := r.app.FindRecordById("time_slots", slot.ID)
	if err != nil {
		return err
	}

	record.Set("current_bookings", slot.CurrentBookings)
	record.Set("is_booked", slot.IsBooked)

	return r.app.Save(record)
}
