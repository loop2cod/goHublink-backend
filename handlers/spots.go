package handlers

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

const spotIDLength = 6

var alphaIDPattern = regexp.MustCompile(`^[A-Za-z]{6}$`)

type spotRequest struct {
	Name          string   `json:"name" binding:"required"`
	Latitude      *float64 `json:"latitude" binding:"required"`
	Longitude     *float64 `json:"longitude" binding:"required"`
	InchargeName  string   `json:"incharge_name"`
	InchargePhone string   `json:"incharge_phone"`
	IDCardType    string   `json:"idcard_type"`
	IDCardName    string   `json:"idcard_name"`
	IDCardDOB     string   `json:"idcard_dob"`
	IDCardNumber  string   `json:"idcard_number"`
	ID            string   `json:"id"`
}

func CreateSpot(c *gin.Context) {
	var req spotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	spotID := strings.ToUpper(strings.TrimSpace(req.ID))
	if spotID != "" && !alphaIDPattern.MatchString(spotID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be 6 alphabetic characters"})
		return
	}
	if spotID == "" {
		var err error
		spotID, err = generateAlphaID(spotIDLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate spot id"})
			return
		}
	}

	spot := models.Spot{
		ID:            spotID,
		Name:          req.Name,
		Latitude:      *req.Latitude,
		Longitude:     *req.Longitude,
		InchargeName:  strings.TrimSpace(req.InchargeName),
		InchargePhone: strings.TrimSpace(req.InchargePhone),
		IDCardType:    strings.TrimSpace(req.IDCardType),
		IDCardName:    strings.TrimSpace(req.IDCardName),
		IDCardDOB:     strings.TrimSpace(req.IDCardDOB),
		IDCardNumber:  strings.TrimSpace(req.IDCardNumber),
	}

	if err := db.DB.Create(&spot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": spot.ID, "name": spot.Name})
}

func ListSpots(c *gin.Context) {
	var spots []models.Spot
	if err := db.DB.Order("created_at DESC").Find(&spots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"spots": spots})
}

func GetSpot(c *gin.Context) {
	var spot models.Spot
	if err := db.DB.Where("id = ?", c.Param("id")).First(&spot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "spot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, spot)
}

func generateAlphaID(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		buf[i] = charset[n.Int64()]
	}
	return string(buf), nil
}