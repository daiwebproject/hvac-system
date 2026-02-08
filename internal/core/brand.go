package core

// Brand represents a business entity/tenant in the system
type Brand struct {
	Id          string `json:"id" db:"id"`
	Slug        string `json:"slug" db:"slug"` // URL slug (e.g., "hung", "pham")
	CompanyName string `json:"company_name" db:"company_name"`
	Active      bool   `json:"active" db:"active"`

	// Branding
	Logo string `json:"logo" db:"logo"`
	Icon string `json:"icon" db:"icon"`

	// Contact
	Hotline string `json:"hotline" db:"hotline"`
	Address string `json:"address" db:"address"`
	Email   string `json:"email" db:"email"`
	Socials string `json:"socials" db:"socials"` // JSON: {zalo, facebook, etc}

	// SEO
	SeoTitle       string `json:"seo_title" db:"seo_title"`
	SeoDescription string `json:"seo_description" db:"seo_description"`
	SeoKeywords    string `json:"seo_keywords" db:"seo_keywords"`

	// Hero Section
	HeroTitle    string `json:"hero_title" db:"hero_title"`
	HeroSubtitle string `json:"hero_subtitle" db:"hero_subtitle"`
	HeroImage    string `json:"hero_image" db:"hero_image"`
	HeroCtaText  string `json:"hero_cta_text" db:"hero_cta_text"`
	HeroCtaLink  string `json:"hero_cta_link" db:"hero_cta_link"`
	WelcomeText  string `json:"welcome_text" db:"welcome_text"`

	// Banking
	BankBin     string `json:"bank_bin" db:"bank_bin"`
	BankAccount string `json:"bank_account" db:"bank_account"`
	BankOwner   string `json:"bank_owner" db:"bank_owner"`
	QrTemplate  string `json:"qr_template" db:"qr_template"`

	// Meta
	Created string `json:"created" db:"created"`
	Updated string `json:"updated" db:"updated"`
}
