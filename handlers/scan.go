package handlers

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

const (
	scanTokenLength = 6
	whatsAppNumber  = "917560845014"
)

func QRRedirect(c *gin.Context) {
	spotID := c.Param("spot_id")

	var spot models.Spot
	if err := db.DB.Where("id = ?", spotID).First(&spot).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "spot not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token, err := generateScanToken(scanTokenLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate scan token"})
		return
	}

	scan := models.Scan{
		ScanToken:    token,
		SpotID:       spotID,
		IPAddress:    clientIP(c),
		UserAgent:    c.GetHeader("User-Agent"),
		DeviceType:   detectDeviceType(c.GetHeader("User-Agent")),
		City:         "",
		ScannedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		Status:       models.ScanStatusPending,
		PhoneNumber:  nil,
		CustomerName: nil,
	}

	if err := db.DB.Create(&scan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	message := formatWhatsAppMessage(spot.Name, token)
	waURL := "https://wa.me/" + whatsAppNumber + "?text=" + url.QueryEscape(message)
	c.Redirect(http.StatusFound, waURL)
}

func formatWhatsAppMessage(spotName, token string) string {
	return "Hi! I just grabbed a spot at " + spotName + ".\n" +
		"My token is: " + token
}

func ListScans(c *gin.Context) {
	var scans []models.Scan

	status := strings.TrimSpace(c.Query("status"))
	spotID := strings.TrimSpace(c.Query("spot_id"))
	token := strings.TrimSpace(c.Query("token"))

	query := db.DB.Order("scanned_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if spotID != "" {
		query = query.Where("spot_id = ?", spotID)
	}
	if token != "" {
		query = query.Where("scan_token ILIKE ?", "%"+token+"%")
	}

	if err := query.Find(&scans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scans": scans})
}

func GetScan(c *gin.Context) {
	var scan models.Scan
	if err := db.DB.Where("id = ?", c.Param("id")).First(&scan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "scan not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, scan)
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if len(ip) > 45 {
		return ip[:45]
	}
	return ip
}

func detectDeviceType(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case lower == "":
		return "desktop"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad") || strings.Contains(lower, "ios"):
		return "ios"
	case strings.Contains(lower, "android"):
		return "android"
	default:
		return "desktop"
	}
}

func generateScanToken(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
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
