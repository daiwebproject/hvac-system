package core

// Settings represents the system configuration
type Settings struct {
	Id          string `json:"id" db:"id"`
	CompanyName string `json:"company_name" db:"company_name"`
	Logo        string `json:"logo" db:"logo"`
	Hotline     string `json:"hotline" db:"hotline"`

	BankBin     string `json:"bank_bin" db:"bank_bin"`
	BankAccount string `json:"bank_account" db:"bank_account"`
	BankOwner   string `json:"bank_owner" db:"bank_owner"`
	QrTemplate  string `json:"qr_template" db:"qr_template"`

	LicenseKey string `json:"license_key" db:"license_key"`
	ExpiryDate string `json:"expiry_date" db:"expiry_date"`
}
