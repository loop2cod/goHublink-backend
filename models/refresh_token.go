package models

import "time"

type RefreshToken struct {
	ID          uint       `gorm:"primaryKey" json:"-"`
	AdminID     uint       `gorm:"not null;index" json:"-"`
	PhoneNumber string     `gorm:"size:20;default:'';index" json:"-"`
	TokenHash   string     `gorm:"uniqueIndex;size:64;not null" json:"-"`
	ExpiresAt   time.Time  `gorm:"not null" json:"-"`
	RevokedAt   *time.Time `gorm:"index" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
}