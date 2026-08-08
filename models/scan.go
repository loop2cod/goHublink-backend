package models

import "time"

type ScanStatus string

const (
	ScanStatusPending ScanStatus = "pending"
	ScanStatusMatched ScanStatus = "matched"
	ScanStatusExpired ScanStatus = "expired"
)

type Scan struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	ScanToken    string     `gorm:"uniqueIndex;size:10;not null" json:"scan_token"`
	SpotID       string     `gorm:"size:6" json:"spot_id"`
	IPAddress    string     `gorm:"size:45" json:"ip_address"`
	UserAgent    string     `gorm:"type:text" json:"user_agent"`
	DeviceType   string     `gorm:"size:20" json:"device_type"`
	City         string     `gorm:"size:50" json:"city"`
	ScannedAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"scanned_at"`
	ExpiresAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP + INTERVAL '10 MINUTE'" json:"expires_at"`
	Status       ScanStatus `gorm:"type:scan_status;default:'pending'" json:"status"`
	PhoneNumber  *string    `gorm:"size:20" json:"phone_number,omitempty"`
	CustomerName *string    `gorm:"size:100" json:"customer_name,omitempty"`
}
