package core

// Settings represents the system configuration
type Settings struct {
	Id          string `json:"id" db:"id"`
	CompanyName string `json:"company_name" db:"company_name"`
	Logo        string `json:"logo" db:"logo"`
	Hotline     string `json:"hotline" db:"hotline"`

	// Bank Info
	BankBin     string `json:"bank_bin" db:"bank_bin"`
	BankAccount string `json:"bank_account" db:"bank_account"`
	BankOwner   string `json:"bank_owner" db:"bank_owner"`
	QrTemplate  string `json:"qr_template" db:"qr_template"`

	// License
	LicenseKey string `json:"license_key" db:"license_key"`
	ExpiryDate string `json:"expiry_date" db:"expiry_date"`

	// [NEW] SEO Configuration
	SeoTitle       string `json:"seo_title" db:"seo_title"`
	SeoDescription string `json:"seo_description" db:"seo_description"`
	SeoKeywords    string `json:"seo_keywords" db:"seo_keywords"`

	// [NEW] Hero Section Configuration
	HeroTitle    string `json:"hero_title" db:"hero_title"`
	HeroSubtitle string `json:"hero_subtitle" db:"hero_subtitle"`
	HeroImage    string `json:"hero_image" db:"hero_image"`
	HeroCtaText  string `json:"hero_cta_text" db:"hero_cta_text"`
	HeroCtaLink  string `json:"hero_cta_link" db:"hero_cta_link"`
	WelcomeText  string `json:"welcome_text" db:"welcome_text"`

	// [NEW] Admin Push Tokens
	AdminFCMTokens []string `json:"admin_fcm_tokens" db:"admin_fcm_tokens"`
}
