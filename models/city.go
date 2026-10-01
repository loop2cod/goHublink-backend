package models

import "time"

type City struct {
	ID              string     `gorm:"primaryKey;size:64" json:"id"`
	Slug            string     `gorm:"size:100;not null;uniqueIndex:idx_cities_slug" json:"slug"`
	Name            string     `gorm:"not null" json:"name"`
	Region          string     `gorm:"size:50;not null;default:'South India'" json:"region"`
	State           string     `gorm:"size:100;default:''" json:"state"`
	Country         string     `gorm:"size:50;default:'India'" json:"country"`
	Latitude        float64    `gorm:"not null" json:"latitude"`
	Longitude       float64    `gorm:"not null" json:"longitude"`
	BoundaryGeoJSON string     `gorm:"type:text" json:"boundary_geojson"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	Divisions       []Division `gorm:"foreignKey:CityID;constraint:OnDelete:CASCADE" json:"divisions,omitempty"`
	DivisionCount   int        `gorm:"-" json:"division_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Division struct {
	ID             string    `gorm:"primaryKey;size:64" json:"id"`
	CityID         string    `gorm:"index;size:64;not null" json:"city_id"`
	Name           string    `gorm:"not null" json:"name"`
	Code           string    `gorm:"size:50" json:"code"`
	CenterLat      float64   `gorm:"not null" json:"center_lat"`
	CenterLng      float64   `gorm:"not null" json:"center_lng"`
	PolygonGeoJSON string    `gorm:"type:text;not null" json:"polygon_geojson"`
	Color          string    `gorm:"size:20;default:''" json:"color"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	SortOrder      int       `gorm:"default:0" json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
