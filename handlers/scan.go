package handlers

import (
	"crypto/rand"
	"encoding/json"
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
	whatsAppNumber  = "918891730090"
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

	ua := c.GetHeader("User-Agent")
	dev := parseUserAgent(ua)
	ip := forwardedIP(c)
	referrer := c.GetHeader("Referer")

	headers := map[string]string{}
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = strings.Join(values, ", ")
		}
	}
	headerJSON, _ := json.Marshal(headers)

	scan := models.Scan{
		ScanToken:      token,
		SpotID:         spotID,
		IPAddress:      clientIP(c),
		IPv6Address:    netIPFromForwarded(c),
		UserAgent:      ua,
		DeviceType:     dev.Type,
		DeviceBrand:    dev.Brand,
		DeviceModel:    dev.Model,
		OSName:         dev.OSName,
		OSVersion:      dev.OSVersion,
		BrowserName:    dev.BrowserName,
		BrowserVersion: dev.BrowserVersion,
		Language:       firstNonEmpty(c.GetHeader("Accept-Language")),
		Referrer:       referrer,
		ReferrerHost:   extractHost(referrer),
		UTMSource:      c.Query("utm_source"),
		UTMMedium:      c.Query("utm_medium"),
		UTMCampaign:    c.Query("utm_campaign"),
		RequestHeaders: headerJSON,
		ScannedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(10 * time.Minute),
		Status:         models.ScanStatusPending,
		PhoneNumber:    nil,
		CustomerName:   nil,
	}

	if err := db.DB.Create(&scan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich with IP geolocation + network info asynchronously
	if ip != "" {
		go enrichScanWithGeo(scan.ID, ip)
	}

	message := formatWhatsAppMessage(token)
	waURL := "https://wa.me/" + whatsAppNumber + "?text=" + url.QueryEscape(message)
	c.Redirect(http.StatusFound, waURL)
}

func formatWhatsAppMessage(token string) string {
	return "Claim it: " + token
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

func generateScanToken(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
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
