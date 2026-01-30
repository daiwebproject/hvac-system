package models

type Service struct {
	Id          string  `db:"id" json:"id"`
	Created     string  `db:"created" json:"created"`
	Name        string  `db:"name" json:"name"`
	Description string  `db:"description" json:"description"`
	Price       float64 `db:"price" json:"price"`
	Image       string  `db:"image" json:"image"`
	Active      bool    `db:"active" json:"active"`
	// Thêm trường này:
	Duration int `db:"duration_minutes" json:"duration_minutes"`
}
