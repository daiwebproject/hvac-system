package repository

import (
	"fmt"
	domain "hvac-system/internal/core"

	"github.com/pocketbase/pocketbase/core"
)

type BrandRepo struct {
	App core.App
}

func NewBrandRepo(app core.App) *BrandRepo {
	return &BrandRepo{App: app}
}

// GetByID finds a brand by ID (Actually fetches settings for Single Tenant)
func (r *BrandRepo) GetByID(id string) (*domain.Brand, error) {
	// In Single Tenant: ID might be ignored if we just want THE settings
	// But let's try to find record by ID in settings
	record, err := r.App.FindRecordById("settings", id)
	if err != nil {
		return nil, err
	}
	return r.recordToModel(record), nil
}

// GetBySlug finds a brand by Slug (In Single Tenant, returns Default)
func (r *BrandRepo) GetBySlug(slug string) (*domain.Brand, error) {
	return r.GetDefault()
}

// GetDefault returns the default brand (The Single Settings Record)
func (r *BrandRepo) GetDefault() (*domain.Brand, error) {
	// 1. Try to find the first settings record
	records, err := r.App.FindRecordsByFilter("settings", "", "-created", 1, 0, nil)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("no settings found")
	}
	return r.recordToModel(records[0]), nil
}

// GetByAdminID finds the brand associated with an admin
// In Single Tenant, returns Default
func (r *BrandRepo) GetByAdminID(adminID string) (*domain.Brand, error) {
	return r.GetDefault()
}

// Create is disabled/redirected for Single Tenant (Use Settings Update)
func (r *BrandRepo) Create(brand *domain.Brand) error {
	return fmt.Errorf("use settings update instead")
}

// Update updates the settings (Brand)
func (r *BrandRepo) Update(brand *domain.Brand) error {
	record, err := r.App.FindRecordById("settings", brand.Id)
	if err != nil {
		return err
	}
	r.modelToRecord(brand, record)
	return r.App.Save(record)
}

// Delete is disabled
func (r *BrandRepo) Delete(id string) error {
	return fmt.Errorf("cannot delete main settings")
}

// Mapper: Record -> Model
func (r *BrandRepo) recordToModel(record *core.Record) *domain.Brand {
	return &domain.Brand{
		Id:          record.Id,
		Slug:        "default", // Hardcoded for Single Tenant
		CompanyName: record.GetString("company_name"),
		Active:      true, // Always active
		Logo:        record.GetString("logo"),
		Icon:        record.GetString("logo"), // Fallback to Logo
		Hotline:     record.GetString("hotline"),
		Address:     "", // Settings doesn't have address yet
		Email:       "", // Settings doesn't have email yet

		SeoTitle:       record.GetString("seo_title"),
		SeoDescription: record.GetString("seo_description"),
		SeoKeywords:    record.GetString("seo_keywords"),

		HeroTitle:    record.GetString("hero_title"),
		HeroSubtitle: record.GetString("hero_subtitle"),
		HeroImage:    record.GetString("hero_image"),
		HeroCtaText:  record.GetString("hero_cta_text"),
		HeroCtaLink:  record.GetString("hero_cta_link"),
		WelcomeText:  record.GetString("welcome_text"),

		BankBin:     record.GetString("bank_bin"),
		BankAccount: record.GetString("bank_account"),
		BankOwner:   record.GetString("bank_owner"),
		QrTemplate:  record.GetString("qr_template"),

		Created: record.GetString("created"),
		Updated: record.GetString("updated"),
	}
}

// Mapper: Model -> Record
func (r *BrandRepo) modelToRecord(brand *domain.Brand, record *core.Record) {
	record.Set("company_name", brand.CompanyName)
	record.Set("hotline", brand.Hotline)
	// Address/Email not in settings yet

	record.Set("seo_title", brand.SeoTitle)
	record.Set("seo_description", brand.SeoDescription)
	record.Set("seo_keywords", brand.SeoKeywords)

	record.Set("hero_title", brand.HeroTitle)
	record.Set("hero_subtitle", brand.HeroSubtitle)
	// Images handled by upload handler usually

	record.Set("hero_cta_text", brand.HeroCtaText)
	record.Set("hero_cta_link", brand.HeroCtaLink)
	record.Set("welcome_text", brand.WelcomeText)

	record.Set("bank_bin", brand.BankBin)
	record.Set("bank_account", brand.BankAccount)
	record.Set("bank_owner", brand.BankOwner)
	record.Set("qr_template", brand.QrTemplate)
}
