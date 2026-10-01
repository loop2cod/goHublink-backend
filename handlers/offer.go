package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

const (
	OfferStatusClaimed  = "claimed"
	OfferStatusRedeemed = "redeemed"
	OfferStatusExpired  = "expired"
)

func ListOffers(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))
	spotID := strings.TrimSpace(c.Query("spot_id"))
	lat := parseFloat(c.Query("lat"))
	lng := parseFloat(c.Query("lng"))

	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	now := time.Now()
	query := db.DB.Model(&models.Offer{}).
		Where("is_active = ?", true).
		Where("valid_from <= ?", now).
		Where("valid_to >= ?", now)

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if spotID != "" {
		query = query.Where("spot_id = ?", spotID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var offers []models.Offer
	if err := query.
		Order("points_cost ASC, created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&offers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if lat != 0 && lng != 0 {
		sortOffersByDistance(offers, lat, lng)
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"offers":      offers,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func GetOffer(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offer id is required"})
		return
	}

	var offer models.Offer
	if err := db.DB.Where("id = ?", id).First(&offer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "offer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, offer)
}

func ClaimOffer(c *gin.Context) {
	phoneNumber, exists := c.Get("phone_number")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	phone := phoneNumber.(string)

	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offer id is required"})
		return
	}

	var offer models.Offer
	if err := db.DB.Where("id = ?", id).First(&offer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "offer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !offer.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offer is not active"})
		return
	}

	now := time.Now()
	if now.Before(offer.ValidFrom) || now.After(offer.ValidTo) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offer is not currently valid"})
		return
	}

	var existing models.ClaimedOffer
	err := db.DB.Where("offer_id = ? AND phone_number = ? AND status = ?", offer.ID, phone, OfferStatusClaimed).
		First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "offer already claimed"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var claimed *models.ClaimedOffer

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if offer.PointsCost > 0 {
			wallet, txErr := getWalletForPhoneNumber(tx, phone)
			if txErr != nil {
				return txErr
			}
			if wallet.Balance < offer.PointsCost {
				return errors.New("insufficient points balance")
			}

			wallet.Balance -= offer.PointsCost
			if txErr := tx.Model(wallet).Update("balance", wallet.Balance).Error; txErr != nil {
				return txErr
			}
			if txErr := applyWalletTier(tx, wallet); txErr != nil {
				return txErr
			}

			transaction := models.WalletTransaction{
				PhoneNumber: phone,
				Amount:      offer.PointsCost,
				Type:        WalletTxTypeDebit,
				Reason:      "claimed offer: " + offer.Title,
				ReferenceID: id,
			}
			if txErr := tx.Create(&transaction).Error; txErr != nil {
				return txErr
			}
		}

		claimed = &models.ClaimedOffer{
			OfferID:     offer.ID,
			PhoneNumber: phone,
			Status:      OfferStatusClaimed,
		}
		return tx.Create(claimed).Error
	})
	if err != nil {
		if err.Error() == "insufficient points balance" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient points balance"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, claimed)
}

func ListClaimedOffers(c *gin.Context) {
	phoneNumber, exists := c.Get("phone_number")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	phone := phoneNumber.(string)

	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}

	query := db.DB.Model(&models.ClaimedOffer{}).
		Where("phone_number = ?", phone).
		Order("claimed_at DESC")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var claimed []models.ClaimedOffer
	if err := query.
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&claimed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offerIDs := make([]uint, 0, len(claimed))
	for _, cl := range claimed {
		offerIDs = append(offerIDs, cl.OfferID)
	}

	offersByID := map[uint]models.Offer{}
	if len(offerIDs) > 0 {
		var offers []models.Offer
		if err := db.DB.Where("id IN ?", offerIDs).Find(&offers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, o := range offers {
			offersByID[o.ID] = o
		}
	}

	items := make([]gin.H, 0, len(claimed))
	for _, cl := range claimed {
		item := gin.H{
			"id":         cl.ID,
			"offer_id":   cl.OfferID,
			"claimed_at": cl.ClaimedAt,
			"status":     cl.Status,
		}
		if cl.RedeemedAt != nil {
			item["redeemed_at"] = cl.RedeemedAt
		}
		if offer, ok := offersByID[cl.OfferID]; ok {
			item["offer"] = offer
		}
		items = append(items, item)
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"claimed_offers": items,
		"total":          total,
		"page":           page,
		"limit":          limit,
		"total_pages":    totalPages,
	})
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	clamped := math.Min(1, math.Max(0, a))
	c := 2 * math.Atan2(math.Sqrt(clamped), math.Sqrt(1-clamped))

	return earthRadius * c
}

func sortOffersByDistance(offers []models.Offer, lat, lng float64) {
	type offerWithDistance struct {
		offer    models.Offer
		distance float64
	}

	spots := make([]string, 0, len(offers))
	for _, o := range offers {
		spots = append(spots, o.SpotID)
	}

	spotCoords := map[string]struct{ lat, lng float64 }{}
	if len(spots) > 0 {
		var spotList []models.Spot
		if err := db.DB.Where("id IN ?", spots).Find(&spotList).Error; err == nil {
			for _, s := range spotList {
				spotCoords[s.ID] = struct{ lat, lng float64 }{s.Latitude, s.Longitude}
			}
		}
	}

	withDist := make([]offerWithDistance, 0, len(offers))
	for _, o := range offers {
		d := math.MaxFloat64
		if coord, ok := spotCoords[o.SpotID]; ok {
			d = haversineDistance(lat, lng, coord.lat, coord.lng)
		}
		withDist = append(withDist, offerWithDistance{offer: o, distance: d})
	}

	for i := 1; i < len(withDist); i++ {
		for j := i; j > 0 && withDist[j].distance < withDist[j-1].distance; j-- {
			withDist[j], withDist[j-1] = withDist[j-1], withDist[j]
		}
	}

	for i := range withDist {
		offers[i] = withDist[i].offer
	}
}
