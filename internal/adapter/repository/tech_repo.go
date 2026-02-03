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
		Email:    record.Email(),
		Avatar:   record.GetString("avatar"),
		Active:   record.GetBool("active"),
		Verified: record.GetBool("verified"),
		FCMToken: record.GetString("fcm_token"), // [NEW]
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

func (r *PBTechnicianRepo) GetAll() ([]*core.Technician, error) {
	records, err := r.app.FindRecordsByFilter("technicians", "", "+created", 500, 0, nil)
	if err != nil {
		return nil, err
	}
	var techs []*core.Technician
	for _, rec := range records {
		techs = append(techs, r.toDomain(rec))
	}
	return techs, nil
}

func (r *PBTechnicianRepo) Create(tech *core.Technician, password string) error {
	collection, err := r.app.FindCollectionByNameOrId("technicians")
	if err != nil {
		return err
	}

	record := pbCore.NewRecord(collection)
	record.Set("email", tech.Email)
	record.Set("name", tech.Name)
	record.Set("verified", tech.Verified)
	record.Set("active", tech.Active)
	record.SetPassword(password)

	return r.app.Save(record)
}

func (r *PBTechnicianRepo) Update(tech *core.Technician) error {
	record, err := r.app.FindRecordById("technicians", tech.ID)
	if err != nil {
		return err
	}

	record.Set("email", tech.Email)
	record.Set("name", tech.Name)
	record.Set("active", tech.Active)
	// Don't update password here

	return r.app.Save(record)
}

func (r *PBTechnicianRepo) SetPassword(id, password string) error {
	record, err := r.app.FindRecordById("technicians", id)
	if err != nil {
		return err
	}
	record.SetPassword(password)
	return r.app.Save(record)
}

func (r *PBTechnicianRepo) ToggleActive(id string) error {
	record, err := r.app.FindRecordById("technicians", id)
	if err != nil {
		return err
	}
	current := record.GetBool("active")
	record.Set("active", !current)
	return r.app.Save(record)
}
