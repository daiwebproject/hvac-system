package repository

import (
	domain "hvac-system/internal/core"
	"log"

	"errors"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type SettingsRepo struct {
	pb *pocketbase.PocketBase
}

func NewSettingsRepo(pb *pocketbase.PocketBase) *SettingsRepo {
	return &SettingsRepo{pb: pb}
}

// GetSettings fetches the single settings record.
// If it doesn't exist (shouldn't happen due to migration), it returns a default object.
func (r *SettingsRepo) GetSettings() (*domain.Settings, error) {
	// 1. Fetch the first record from 'settings' collection
	records, err := r.pb.FindRecordsByFilter(
		"settings",
		"1=1", // No filter, just get one
		"-created",
		1,
		0,
		nil,
	)

	if err != nil || len(records) == 0 {
		log.Printf("⚠️ Warning: Could not fetch settings from DB: %v. Using hardcoded defaults.", err)
		return &domain.Settings{
			CompanyName: "HVAC System (Default)",
			BankBin:     "970422",
			BankAccount: "0333666999",
			BankOwner:   "DEFAULT OWNER",
		}, nil
	}

	rec := records[0]

	// 2. Map to Struct
	return &domain.Settings{
		Id:          rec.Id,
		CompanyName: rec.GetString("company_name"),
		Logo:        rec.GetString("logo"),
		Hotline:     rec.GetString("hotline"),
		BankBin:     rec.GetString("bank_bin"),
		BankAccount: rec.GetString("bank_account"),
		BankOwner:   rec.GetString("bank_owner"),
		QrTemplate:  rec.GetString("qr_template"),
		LicenseKey:  rec.GetString("license_key"),
		ExpiryDate:  rec.GetString("expiry_date"),
	}, nil
}

// GetSettingsRecord returns the raw PocketBase record for updating
func (r *SettingsRepo) GetSettingsRecord() (*core.Record, error) {
	records, err := r.pb.FindRecordsByFilter(
		"settings",
		"1=1",
		"-created",
		1,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.New("settings record not found")
	}
	return records[0], nil
}
