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
	status := strings.TrimSpace(c.Query("status"))
	spotID := strings.TrimSpace(c.Query("spot_id"))
	token := strings.TrimSpace(c.Query("token"))
	device := strings.ToLower(strings.TrimSpace(c.Query("device")))
	q := strings.TrimSpace(c.Query("q"))

	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 10)
	if limit > 100 {
		limit = 100
	}

	query := db.DB.Model(&models.Scan{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if spotID != "" {
		query = query.Where("spot_id = ?", spotID)
	}
	if token != "" {
		query = query.Where("scan_token ILIKE ?", "%"+token+"%")
	}
	if device != "" {
		query = query.Where("LOWER(device_type) = ?", device)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"scan_token ILIKE ? OR spot_id ILIKE ? OR ip_address ILIKE ? OR "+
				"COALESCE(customer_name,'') ILIKE ? OR COALESCE(city,'') ILIKE ? OR "+
				"COALESCE(region,'') ILIKE ? OR COALESCE(country_name,'') ILIKE ? OR "+
				"LOWER(device_type) ILIKE ? OR COALESCE(os_name,'') ILIKE ? OR "+
				"COALESCE(browser_name,'') ILIKE ? OR COALESCE(isp,'') ILIKE ?",
			like, like, like, like, like, like, like,
			strings.ToLower(q)+"%", like, like, like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var scans []models.Scan
	if err := query.
		Order("scanned_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&scans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"scans":       scans,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// ScanStats returns aggregate counts and the distinct list of device types
// across all scans, used for the listing summary cards and device filter
// (independent of the current filters/pagination).
func ScanStats(c *gin.Context) {
	var total, matched, pending, expired int64

	if err := db.DB.Model(&models.Scan{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := db.DB.Model(&models.Scan{}).Where("status = ?", models.ScanStatusMatched).Count(&matched).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := db.DB.Model(&models.Scan{}).Where("status = ?", models.ScanStatusPending).Count(&pending).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := db.DB.Model(&models.Scan{}).Where("status = ?", models.ScanStatusExpired).Count(&expired).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var devices []string
	if err := db.DB.Model(&models.Scan{}).
		Distinct().
		Where("device_type <> ''").
		Order("device_type ASC").
		Pluck("device_type", &devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"matched": matched,
		"pending": pending,
		"expired": expired,
		"devices": devices,
	})
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

// GetScanByToken fetches a single scan by its scan token (e.g. "5Z2J6U").
func GetScanByToken(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	var scan models.Scan
	if err := db.DB.Where("scan_token = ?", token).First(&scan).Error; err != nil {
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
