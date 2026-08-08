package models

import "time"

type Customer struct {
	PhoneNumber         string    `gorm:"primaryKey;size:20" json:"phone_number"`
	Name                string    `gorm:"size:100" json:"name"`
	FirstScanAt         time.Time `json:"first_scan_at"`
	LastActive          time.Time `json:"last_active"`
	PreferredLocationID int       `gorm:"type:int;index" json:"preferred_location_id"`
}
