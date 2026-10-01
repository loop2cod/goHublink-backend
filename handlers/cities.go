package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

type divisionInput struct {
	ID             string  `json:"id"`
	Name           string  `json:"name" binding:"required"`
	Code           string  `json:"code"`
	CenterLat      float64 `json:"center_lat" binding:"required"`
	CenterLng      float64 `json:"center_lng" binding:"required"`
	PolygonGeoJSON string  `json:"polygon_geojson" binding:"required"`
	Color          string  `json:"color"`
	IsActive       *bool   `json:"is_active"`
	SortOrder      int     `json:"sort_order"`
}

type createCityRequest struct {
	ID              string          `json:"id"`
	Name            string          `json:"name" binding:"required"`
	Region          string          `json:"region"`
	State           string          `json:"state"`
	Country         string          `json:"country"`
	Latitude        float64         `json:"latitude" binding:"required"`
	Longitude       float64         `json:"longitude" binding:"required"`
	BoundaryGeoJSON string          `json:"boundary_geojson"`
	IsActive        *bool           `json:"is_active"`
	Divisions       []divisionInput `json:"divisions"`
}

type updateCityRequest struct {
	Name            *string  `json:"name"`
	Region          *string  `json:"region"`
	State           *string  `json:"state"`
	Country         *string  `json:"country"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	BoundaryGeoJSON *string  `json:"boundary_geojson"`
	IsActive        *bool    `json:"is_active"`
}

type updateDivisionRequest struct {
	Name           *string  `json:"name"`
	Code           *string  `json:"code"`
	CenterLat      *float64 `json:"center_lat"`
	CenterLng      *float64 `json:"center_lng"`
	PolygonGeoJSON *string  `json:"polygon_geojson"`
	Color          *string  `json:"color"`
	IsActive       *bool    `json:"is_active"`
	SortOrder      *int     `json:"sort_order"`
}

func ListCities(c *gin.Context) {
	var cities []models.City
	if err := db.DB.Order("name ASC").Find(&cities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type divisionCountResult struct {
		CityID string
		Count  int
	}
	var counts []divisionCountResult
	db.DB.Model(&models.Division{}).Select("city_id, count(*) as count").Group("city_id").Scan(&counts)

	countMap := make(map[string]int)
	for _, cnt := range counts {
		countMap[cnt.CityID] = cnt.Count
	}

	for i := range cities {
		cities[i].DivisionCount = countMap[cities[i].ID]
	}

	c.JSON(http.StatusOK, gin.H{"cities": cities})
}

func GetCity(c *gin.Context) {
	id := c.Param("id")
	var city models.City
	if err := db.DB.Preload("Divisions", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("divisions.sort_order ASC, divisions.name ASC")
	}).Where("id = ?", id).First(&city).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	city.DivisionCount = len(city.Divisions)
	c.JSON(http.StatusOK, city)
}

func CreateCity(c *gin.Context) {
	var req createCityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cityID := strings.ToLower(strings.TrimSpace(req.ID))
	if cityID == "" {
		cityID = slugify(req.Name)
	}
	if cityID == "" {
		cityID = generateRandomHex(8)
	}

	// Upsert semantics: the admin UI re-POSTs the same city ID every time
	// the operator re-partitions a city. If the city already exists, update
	// it in place and replace its divisions instead of failing on the PK.
	var existingCity models.City
	isUpdate := db.DB.Where("id = ?", cityID).First(&existingCity).Error == nil

	// The cities table requires a unique, non-null slug. Derive it from the
	// city ID (already slugified) and disambiguate on collision. Only needed
	// for fresh inserts; updates keep the existing slug.
	slug := cityID
	if !isUpdate {
		for i := 0; i < 5; i++ {
			var existing int64
			if err := db.DB.Model(&models.City{}).Where("slug = ?", slug).Count(&existing).Error; err != nil {
				break
			}
			if existing == 0 {
				break
			}
			slug = cityID + "-" + generateRandomHex(4)
		}
	}

	region := strings.TrimSpace(req.Region)
	if region == "" {
		region = "South India"
	}
	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = "India"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	city := models.City{
		ID:              cityID,
		Slug:            slug,
		Name:            strings.TrimSpace(req.Name),
		Region:          region,
		State:           strings.TrimSpace(req.State),
		Country:         country,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		BoundaryGeoJSON: req.BoundaryGeoJSON,
		IsActive:        isActive,
	}

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if isUpdate {
			existingCity.Name = strings.TrimSpace(req.Name)
			existingCity.Region = region
			existingCity.State = strings.TrimSpace(req.State)
			existingCity.Country = country
			existingCity.Latitude = req.Latitude
			existingCity.Longitude = req.Longitude
			existingCity.BoundaryGeoJSON = req.BoundaryGeoJSON
			existingCity.IsActive = isActive
			if err := tx.Save(&existingCity).Error; err != nil {
				return err
			}
			if err := tx.Where("city_id = ?", cityID).Delete(&models.Division{}).Error; err != nil {
				return err
			}
			city = existingCity
		} else if err := tx.Create(&city).Error; err != nil {
			return err
		}

		for idx, divInput := range req.Divisions {
			divID := strings.TrimSpace(divInput.ID)
			if divID == "" {
				divID = cityID + "-" + slugify(divInput.Name)
				if divID == cityID+"-" {
					divID = cityID + "-" + generateRandomHex(6)
				}
			}

			divIsActive := true
			if divInput.IsActive != nil {
				divIsActive = *divInput.IsActive
			}

			code := strings.TrimSpace(divInput.Code)
			if code == "" {
				code = strings.ToUpper(slugify(divInput.Name))
			}

			sortOrder := divInput.SortOrder
			if sortOrder == 0 {
				sortOrder = idx + 1
			}

			division := models.Division{
				ID:             divID,
				CityID:         cityID,
				Name:           strings.TrimSpace(divInput.Name),
				Code:           code,
				CenterLat:      divInput.CenterLat,
				CenterLng:      divInput.CenterLng,
				PolygonGeoJSON: divInput.PolygonGeoJSON,
				Color:          divInput.Color,
				IsActive:       divIsActive,
				SortOrder:      sortOrder,
			}

			if err := tx.Create(&division).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := http.StatusCreated
	if isUpdate {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"id": city.ID, "name": city.Name, "divisions_count": len(req.Divisions)})
}

func UpdateCity(c *gin.Context) {
	id := c.Param("id")
	var city models.City
	if err := db.DB.Where("id = ?", id).First(&city).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req updateCityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		city.Name = strings.TrimSpace(*req.Name)
	}
	if req.Region != nil {
		city.Region = strings.TrimSpace(*req.Region)
	}
	if req.State != nil {
		city.State = strings.TrimSpace(*req.State)
	}
	if req.Country != nil {
		city.Country = strings.TrimSpace(*req.Country)
	}
	if req.Latitude != nil {
		city.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		city.Longitude = *req.Longitude
	}
	if req.BoundaryGeoJSON != nil {
		city.BoundaryGeoJSON = *req.BoundaryGeoJSON
	}
	if req.IsActive != nil {
		city.IsActive = *req.IsActive
	}

	if err := db.DB.Save(&city).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, city)
}

func DeleteCity(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("id = ?", id).Delete(&models.City{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "city and its divisions deleted"})
}

func CreateDivision(c *gin.Context) {
	cityID := c.Param("id")
	var city models.City
	if err := db.DB.Where("id = ?", cityID).First(&city).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
		return
	}

	var req divisionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	divID := strings.TrimSpace(req.ID)
	if divID == "" {
		divID = cityID + "-" + slugify(req.Name)
		if divID == cityID+"-" {
			divID = cityID + "-" + generateRandomHex(6)
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = strings.ToUpper(slugify(req.Name))
	}

	div := models.Division{
		ID:             divID,
		CityID:         cityID,
		Name:           strings.TrimSpace(req.Name),
		Code:           code,
		CenterLat:      req.CenterLat,
		CenterLng:      req.CenterLng,
		PolygonGeoJSON: req.PolygonGeoJSON,
		Color:          req.Color,
		IsActive:       isActive,
		SortOrder:      req.SortOrder,
	}

	if err := db.DB.Create(&div).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, div)
}

func UpdateDivision(c *gin.Context) {
	divisionID := c.Param("division_id")
	var div models.Division
	if err := db.DB.Where("id = ?", divisionID).First(&div).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "division not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req updateDivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		div.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		div.Code = strings.TrimSpace(*req.Code)
	}
	if req.CenterLat != nil {
		div.CenterLat = *req.CenterLat
	}
	if req.CenterLng != nil {
		div.CenterLng = *req.CenterLng
	}
	if req.PolygonGeoJSON != nil {
		div.PolygonGeoJSON = *req.PolygonGeoJSON
	}
	if req.Color != nil {
		div.Color = *req.Color
	}
	if req.IsActive != nil {
		div.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		div.SortOrder = *req.SortOrder
	}

	if err := db.DB.Save(&div).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, div)
}

func DeleteDivision(c *gin.Context) {
	divisionID := c.Param("division_id")
	if err := db.DB.Where("id = ?", divisionID).Delete(&models.Division{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "division deleted"})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' || r == ',' {
			if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "-") {
				sb.WriteRune('-')
			}
		}
	}
	return strings.Trim(sb.String(), "-")
}

func generateRandomHex(n int) string {
	bytes := make([]byte, (n+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return "x1"
	}
	h := hex.EncodeToString(bytes)
	if len(h) > n {
		return h[:n]
	}
	return h
}
