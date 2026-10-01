package models

import "time"

type Spot struct {
	ID            string    `gorm:"primaryKey;size:6;type:char(6)" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	Latitude      float64   `gorm:"not null" json:"latitude"`
	Longitude     float64   `gorm:"not null" json:"longitude"`
	InchargeName  string    `gorm:"default:''" json:"incharge_name"`
	InchargePhone string    `gorm:"default:''" json:"incharge_phone"`
	IDCardType    string    `gorm:"default:''" json:"idcard_type"`
	IDCardName    string    `gorm:"default:''" json:"idcard_name"`
	IDCardDOB     string    `gorm:"default:''" json:"idcard_dob"`
	IDCardNumber  string    `gorm:"default:''" json:"idcard_number"`
	CityID        string    `gorm:"size:64;default:''" json:"city_id"`
	DivisionID    string    `gorm:"size:64;default:''" json:"division_id"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}