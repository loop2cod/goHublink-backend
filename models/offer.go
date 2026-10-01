package models

import "time"

type Offer struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SpotID         string    `gorm:"size:6;not null;index" json:"spot_id"`
	Title          string    `gorm:"not null" json:"title"`
	Description    string    `gorm:"type:text" json:"description"`
	Category       string    `gorm:"size:50;not null" json:"category"` // "food", "cafe", "nightlife", "events", "gym", "salon"
	DiscountType   string    `gorm:"size:20;not null" json:"discount_type"` // "percentage", "flat", "bogo"
	DiscountValue  int       `gorm:"not null" json:"discount_value"`
	MinSpend       int       `gorm:"default:0" json:"min_spend"`
	PointsCost     int       `gorm:"default:0" json:"points_cost"` // 0 = free claim
	ValidFrom      time.Time `json:"valid_from"`
	ValidTo        time.Time `json:"valid_to"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ClaimedOffer struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	OfferID     uint       `gorm:"not null;index" json:"offer_id"`
	PhoneNumber string     `gorm:"size:20;not null;index" json:"phone_number"`
	ClaimedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"claimed_at"`
	RedeemedAt  *time.Time `json:"redeemed_at,omitempty"`
	Status      string     `gorm:"size:20;default:'claimed'" json:"status"` // "claimed", "redeemed", "expired"
}
