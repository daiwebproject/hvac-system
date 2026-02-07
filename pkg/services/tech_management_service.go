package services

import (
	"errors"
	"hvac-system/internal/core"
)

type TechManagementService struct {
	repo core.TechnicianRepository
}

func NewTechManagementService(repo core.TechnicianRepository) *TechManagementService {
	return &TechManagementService{repo: repo}
}

func (s *TechManagementService) GetAllTechs() ([]*core.Technician, error) {
	return s.repo.GetAll()
}

func (s *TechManagementService) CreateTech(name, email, password string) error {
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	tech := &core.Technician{
		Name:     name,
		Email:    email,
		Active:   true,
		Verified: true,
	}

	return s.repo.Create(tech, password)
}

type UpdateTechInput struct {
	Name           string
	Email          string
	Level          string
	BaseSalary     float64
	CommissionRate float64
	Skills         []string
	ServiceZones   []string
}

func (s *TechManagementService) UpdateTech(id string, input UpdateTechInput) error {
	tech, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	tech.Name = input.Name
	tech.Email = input.Email
	tech.Level = input.Level
	tech.BaseSalary = input.BaseSalary
	tech.CommissionRate = input.CommissionRate
	tech.Skills = input.Skills
	tech.ServiceZones = input.ServiceZones

	return s.repo.Update(tech)
}

func (s *TechManagementService) ResetPassword(id, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return s.repo.SetPassword(id, newPassword)
}

func (s *TechManagementService) ToggleActiveStatus(id string) error {
	return s.repo.ToggleActive(id)
}
