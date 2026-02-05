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
		Id:             rec.Id,
		CompanyName:    rec.GetString("company_name"),
		Logo:           rec.GetString("logo"),
		Hotline:        rec.GetString("hotline"),
		BankBin:        rec.GetString("bank_bin"),
		BankAccount:    rec.GetString("bank_account"),
		BankOwner:      rec.GetString("bank_owner"),
		QrTemplate:     rec.GetString("qr_template"),
		LicenseKey:     rec.GetString("license_key"),
		ExpiryDate:     rec.GetString("expiry_date"),
		SeoTitle:       rec.GetString("seo_title"),
		SeoDescription: rec.GetString("seo_description"),
		SeoKeywords:    rec.GetString("seo_keywords"),
		HeroImage:      rec.GetString("hero_image"),
		HeroTitle:      rec.GetString("hero_title"),
		HeroSubtitle:   rec.GetString("hero_subtitle"),
		HeroCtaText:    rec.GetString("hero_cta_text"),
		HeroCtaLink:    rec.GetString("hero_cta_link"),
		WelcomeText:    rec.GetString("welcome_text"),
		AdminFCMTokens: rec.GetStringSlice("admin_fcm_tokens"), // [NEW]
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

// AddAdminToken appends a new FCM token to the AdminFCMTokens list
func (r *SettingsRepo) AddAdminToken(token string) error {
	record, err := r.GetSettingsRecord()
	if err != nil {
		return err
	}

	// 1. Get current tokens
	currentTokens := record.GetStringSlice("admin_fcm_tokens")

	// 2. Check overlap
	for _, t := range currentTokens {
		if t == token {
			return nil // Already exists
		}
	}

	// 3. Append
	currentTokens = append(currentTokens, token)
	record.Set("admin_fcm_tokens", currentTokens)

	// 4. Save
	return r.pb.Save(record)
}

// RemoveAdminToken removes a token from the AdminFCMTokens list
func (r *SettingsRepo) RemoveAdminToken(token string) error {
	record, err := r.GetSettingsRecord()
	if err != nil {
		return err
	}

	// 1. Get current tokens
	currentTokens := record.GetStringSlice("admin_fcm_tokens")
	newTokens := []string{}

	// 2. Filter out the specific token
	found := false
	for _, t := range currentTokens {
		if t != token {
			newTokens = append(newTokens, t)
		} else {
			found = true
		}
	}

	if !found {
		return nil // Token not found, nothing to do
	}

	// 3. Update and Save
	record.Set("admin_fcm_tokens", newTokens)
	return r.pb.Save(record)
}
