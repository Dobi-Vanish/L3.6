package model

import "time"

type Transaction struct {
	ID          string    `json:"id" db:"id"`
	Type        string    `json:"type" db:"type"`
	Category    string    `json:"category" db:"category"`
	Amount      float64   `json:"amount" db:"amount"`
	Description string    `json:"description" db:"description"`
	Date        time.Time `json:"date" db:"date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
