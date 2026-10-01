package models

import "time"

type Wallet struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"uniqueIndex;size:20;not null" json:"phone_number"`
	Balance     int       `gorm:"default:0" json:"balance"`
	Tier        string    `gorm:"size:20;default:'bronze'" json:"tier"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WalletTransaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"index;size:20;not null" json:"phone_number"`
	Amount      int       `json:"amount"`
	Type        string    `gorm:"size:20;not null" json:"type"` // "credit" or "debit"
	Reason      string    `gorm:"size:100;not null" json:"reason"`
	ReferenceID string    `gorm:"size:100" json:"reference_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
