package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

type geoIPResponse struct {
	Status          string  `json:"status"`
	Country         string  `json:"country"`
	CountryCode     string  `json:"countryCode"`
	Region          string  `json:"regionName"`
	RegionCode      string  `json:"region"`
	City            string  `json:"city"`
	Zip             string  `json:"zip"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	Timezone        string  `json:"timezone"`
	ISP             string  `json:"isp"`
	Org             string  `json:"org"`
	AS              string  `json:"as"`
	Query           string  `json:"query"`
	Proxy           bool    `json:"proxy"`
	Hosting         bool    `json:"hosting"`
}

type deviceInfo struct {
	Type           string
	Brand          string
	Model          string
	OSName         string
	OSVersion      string
	BrowserName    string
	BrowserVersion string
}

func parseUserAgent(ua string) deviceInfo {
	if ua == "" {
		return deviceInfo{Type: "desktop"}
	}

	parsed := useragent.Parse(ua)

	info := deviceInfo{
		Type:           detectDeviceType(ua),
		Model:          parsed.Device,
		OSName:         parsed.OS,
		OSVersion:      parsed.OSVersion,
		BrowserName:    parsed.Name,
		BrowserVersion: parsed.Version,
	}

	switch {
	case parsed.IsAndroid():
		info.Brand = "Android"
	case parsed.IsIOS():
		info.Brand = "Apple"
	case parsed.IsWindows():
		info.Brand = "Microsoft"
	case parsed.IsMacOS():
		info.Brand = "Apple"
	case parsed.IsLinux():
		info.Brand = "Linux"
	}

	if parsed.Mobile {
		info.Type = "mobile"
	} else if parsed.Tablet {
		info.Type = "tablet"
	} else if parsed.Desktop {
		info.Type = "desktop"
	}

	return info
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
	case strings.Contains(lower, "mobile"):
		return "mobile"
	case strings.Contains(lower, "tablet"):
		return "tablet"
	default:
		return "desktop"
	}
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if len(ip) > 45 {
		return ip[:45]
	}
	return ip
}

func forwardedIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	realIP := c.GetHeader("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return clientIP(c)
}

func netIPFromForwarded(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		for _, p := range parts {
			ip := strings.TrimSpace(p)
			if net.ParseIP(ip) != nil && net.ParseIP(ip).To4() == nil {
				return ip
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func extractHost(referrer string) string {
	if referrer == "" {
		return ""
	}
	parts := strings.Split(referrer, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func queryIPVersion(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if parsed.To4() != nil {
		return "IPv4"
	}
	return "IPv6"
}

func lookupGeoIP(ip string) (*geoIPResponse, error) {
	if ip == "" || ip == "::1" || ip == "127.0.0.1" || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		return nil, nil
	}

	url := "http://ip-api.com/json/" + ip + "?fields=status,country,countryCode,regionName,region,city,zip,lat,lon,timezone,isp,org,as,query,proxy,hosting"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var geo geoIPResponse
	if err := json.Unmarshal(body, &geo); err != nil {
		return nil, err
	}

	if geo.Status != "success" {
		return nil, nil
	}

	return &geo, nil
}

func enrichScanWithGeo(scanID uint64, ip string) {
	geo, err := lookupGeoIP(ip)
	if err != nil {
		log.Printf("GeoIP lookup failed for %s: %v", ip, err)
		return
	}
	if geo == nil {
		return
	}

	updates := map[string]interface{}{
		"city":             geo.City,
		"region":           geo.Region,
		"region_code":      geo.RegionCode,
		"country":          geo.CountryCode,
		"country_name":     geo.Country,
		"postal_code":      geo.Zip,
		"latitude":         geo.Lat,
		"longitude":        geo.Lon,
		"timezone":         geo.Timezone,
		"isp":              geo.ISP,
		"network_org":      geo.Org,
		"as_number":        geo.AS,
		"proxy":            geo.Proxy,
		"hosting":          geo.Hosting,
	}

	if err := db.DB.Model(&models.Scan{}).Where("id = ?", scanID).Updates(updates).Error; err != nil {
		log.Printf("Failed to enrich scan %d with geo data: %v", scanID, err)
	}
}
