package models

import (
	"time"

	"gorm.io/datatypes"
)

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
	Status       ScanStatus `gorm:"type:scan_status;default:'pending'" json:"status"`
	ScannedAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"scanned_at"`
	ExpiresAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP + INTERVAL '10 MINUTE'" json:"expires_at"`

	// Network
	IPAddress       string `gorm:"size:45" json:"ip_address"`
	IPv6Address     string `gorm:"size:45" json:"ipv6_address,omitempty"`
	ISP             string `gorm:"size:150" json:"isp,omitempty"`
	ASNumber        string `gorm:"size:50" json:"as_number,omitempty"`
	ASOrganization  string `gorm:"size:150" json:"as_organization,omitempty"`
	ConnectionType  string `gorm:"size:50" json:"connection_type,omitempty"`
	NetworkOrg      string `gorm:"size:150" json:"network_org,omitempty"`
	Proxy           bool   `json:"proxy"`
	Hosting         bool   `json:"hosting"`

	// Device
	UserAgent      string `gorm:"type:text" json:"user_agent"`
	DeviceType     string `gorm:"size:20" json:"device_type"`
	DeviceBrand    string `gorm:"size:50" json:"device_brand,omitempty"`
	DeviceModel    string `gorm:"size:100" json:"device_model,omitempty"`
	OSName         string `gorm:"size:50" json:"os_name,omitempty"`
	OSVersion      string `gorm:"size:50" json:"os_version,omitempty"`
	BrowserName    string `gorm:"size:50" json:"browser_name,omitempty"`
	BrowserVersion string `gorm:"size:50" json:"browser_version,omitempty"`
	Language       string `gorm:"size:255" json:"language,omitempty"`

	// Location
	City          string  `gorm:"size:50" json:"city"`
	Region        string  `gorm:"size:50" json:"region,omitempty"`
	RegionCode    string  `gorm:"size:10" json:"region_code,omitempty"`
	Country       string  `gorm:"size:2" json:"country,omitempty"`
	CountryName   string  `gorm:"size:100" json:"country_name,omitempty"`
	PostalCode    string  `gorm:"size:20" json:"postal_code,omitempty"`
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	LocationAccuracy int   `json:"location_accuracy,omitempty"`

	// Referral
	Referrer     string `gorm:"size:500" json:"referrer,omitempty"`
	ReferrerHost string `gorm:"size:255" json:"referrer_host,omitempty"`
	UTMSource    string `gorm:"size:100" json:"utm_source,omitempty"`
	UTMMedium    string `gorm:"size:100" json:"utm_medium,omitempty"`
	UTMCampaign  string `gorm:"size:100" json:"utm_campaign,omitempty"`

	// Raw
	RequestHeaders datatypes.JSON `gorm:"type:jsonb" json:"request_headers,omitempty"`

	// Customer (set on match)
	PhoneNumber  *string `gorm:"size:20" json:"phone_number,omitempty"`
	CustomerName *string `gorm:"size:100" json:"customer_name,omitempty"`
}
