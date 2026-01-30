package repository

import (
	"hvac-system/internal/core"

	pbCore "github.com/pocketbase/pocketbase/core"
)

type PBTechnicianRepo struct {
	app pbCore.App
}

func NewTechnicianRepo(app pbCore.App) core.TechnicianRepository {
	return &PBTechnicianRepo{app: app}
}

func (r *PBTechnicianRepo) toDomain(record *pbCore.Record) *core.Technician {
	return &core.Technician{
		ID:       record.Id,
		Name:     record.GetString("name"),
		Active:   record.GetBool("active"),
		Verified: record.GetBool("verified"),
	}
}

func (r *PBTechnicianRepo) GetByID(id string) (*core.Technician, error) {
	record, err := r.app.FindRecordById("technicians", id)
	if err != nil {
		return nil, err
	}
	return r.toDomain(record), nil
}

func (r *PBTechnicianRepo) GetAvailable() ([]*core.Technician, error) {
	// This simplified version just gets all active technicians
	// Availability logic (checking assignments) is usually Service layer concern or complex query
	records, err := r.app.FindRecordsByFilter("technicians", "active = true", "name", 100, 0, nil)
	if err != nil {
		return nil, err
	}

	var techs []*core.Technician
	for _, rec := range records {
		techs = append(techs, r.toDomain(rec))
	}
	return techs, nil
}
