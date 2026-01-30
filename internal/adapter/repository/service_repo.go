package repository

import (
	"hvac-system/internal/core"

	pbCore "github.com/pocketbase/pocketbase/core"
)

type PBServiceRepo struct {
	app pbCore.App
}

func NewServiceRepo(app pbCore.App) core.ServiceRepository {
	return &PBServiceRepo{app: app}
}

func (r *PBServiceRepo) GetByID(id string) (*core.Service, error) {
	record, err := r.app.FindRecordById("services", id)
	if err != nil {
		return nil, err
	}

	return &core.Service{
		ID:              record.Id,
		Name:            record.GetString("name"),
		DurationMinutes: record.GetInt("duration_minutes"),
	}, nil
}
