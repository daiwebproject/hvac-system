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
		ID:             record.Id,
		Name:           record.GetString("name"),
		Email:          record.Email(),
		Avatar:         record.GetString("avatar"),
		Active:         record.GetBool("active"),
		Verified:       record.GetBool("verified"),
		FCMToken:       record.GetString("fcm_token"),
		Phone:          record.GetString("phone"),
		Rating:         record.GetFloat("rating"),
		Level:          record.GetString("level"),
		Skills:         record.GetStringSlice("skills"),
		ServiceZones:   record.GetStringSlice("service_zones"),
		BaseSalary:     record.GetFloat("base_salary"),
		CommissionRate: record.GetFloat("commission_rate"),
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
	record.Set("level", tech.Level)
	record.Set("skills", tech.Skills)
	record.Set("service_zones", tech.ServiceZones)
	record.Set("base_salary", tech.BaseSalary)
	record.Set("commission_rate", tech.CommissionRate)
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
	record.Set("level", tech.Level)
	record.Set("skills", tech.Skills)
	record.Set("service_zones", tech.ServiceZones)
	record.Set("base_salary", tech.BaseSalary)
	record.Set("commission_rate", tech.CommissionRate)
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

// UpdateFCMToken updates the FCM token for a technician
func (r *PBTechnicianRepo) UpdateFCMToken(techID, token string) error {
	record, err := r.app.FindRecordById("technicians", techID)
	if err != nil {
		return err
	}
	record.Set("fcm_token", token)
	return r.app.Save(record)
}

// ClearFCMTokenExcept removes the given FCM token from all technicians except the specified one
// This prevents token leakage when a device is shared between technicians
func (r *PBTechnicianRepo) ClearFCMTokenExcept(token, exceptTechID string) error {
	filter := "fcm_token = {:token} && id != {:except}"
	params := map[string]any{"token": token, "except": exceptTechID}
	records, err := r.app.FindRecordsByFilter("technicians", filter, "", 100, 0, params)
	if err != nil {
		return err
	}

	for _, rec := range records {
		rec.Set("fcm_token", "")
		if err := r.app.Save(rec); err != nil {
			return err
		}
	}
	return nil
}

// CountActive returns the number of active technicians
func (r *PBTechnicianRepo) CountActive() (int, error) {
	records, err := r.app.FindRecordsByFilter("technicians", "active = true", "", 0, 0, nil)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}
