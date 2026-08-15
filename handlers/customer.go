package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gohublink/backend/db"
	"gohublink/backend/models"
)

// CustomerListItem is a customer row for the listing, enriched with aggregated
// scan statistics and their most common ("usual") location.
type CustomerListItem struct {
	PhoneNumber       string    `json:"phone_number"`
	Name              string    `json:"name"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
	FirstScanAt       string    `json:"first_scan_at"`
	LastActive        string    `json:"last_active"`
	TotalScans        int64     `json:"total_scans"`
	MatchedScans      int64     `json:"matched_scans"`
	PendingScans      int64     `json:"pending_scans"`
	SpotsVisited      int64     `json:"spots_visited"`
	UsualCity         string    `json:"usual_city,omitempty"`
	UsualRegion       string    `json:"usual_region,omitempty"`
	UsualCountry      string    `json:"usual_country,omitempty"`
}

// ListCustomers returns paginated customers with search and aggregated stats.
func ListCustomers(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 10)
	if limit > 100 {
		limit = 100
	}

	base := db.DB.Model(&models.Customer{})
	if q != "" {
		like := "%" + q + "%"
		base = base.Where(
			"name ILIKE ? OR phone_number ILIKE ?",
			like, like,
		)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var customers []models.Customer
	if err := base.
		Order("last_active DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	phones := make([]string, 0, len(customers))
	for _, cu := range customers {
		phones = append(phones, cu.PhoneNumber)
	}

	stats := customerScanAggregates(phones)

	items := make([]CustomerListItem, 0, len(customers))
	for _, cu := range customers {
		agg := stats[cu.PhoneNumber]
		items = append(items, CustomerListItem{
			PhoneNumber:       cu.PhoneNumber,
			Name:              cu.Name,
			ProfilePictureURL: cu.ProfilePictureURL,
			FirstScanAt:       cu.FirstScanAt.Format("2006-01-02T15:04:05Z07:00"),
			LastActive:        cu.LastActive.Format("2006-01-02T15:04:05Z07:00"),
			TotalScans:        agg.total,
			MatchedScans:      agg.matched,
			PendingScans:      agg.pending,
			SpotsVisited:      agg.spots,
			UsualCity:         agg.usualCity,
			UsualRegion:       agg.usualRegion,
			UsualCountry:      agg.usualCountry,
		})
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"customers":   items,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

type scanAggregate struct {
	total       int64
	matched     int64
	pending     int64
	spots       int64
	usualCity   string
	usualRegion string
	usualCountry string
}

// customerScanAggregates returns, per phone number, scan counts plus the most
// frequent location among that customer's scans.
func customerScanAggregates(phones []string) map[string]*scanAggregate {
	result := map[string]*scanAggregate{}
	if len(phones) == 0 {
		return result
	}

	for _, p := range phones {
		result[p] = &scanAggregate{}
	}

	var scans []models.Scan
	db.DB.
		Select("phone_number, status, spot_id, city, region, country_name").
		Where("phone_number IN ?", phones).
		Find(&scans)

	cityCount := map[string]map[string]int{}
	for _, s := range scans {
		if s.PhoneNumber == nil {
			continue
		}
		phone := *s.PhoneNumber
		agg := result[phone]
		if agg == nil {
			continue
		}
		agg.total++
		switch s.Status {
		case models.ScanStatusMatched:
			agg.matched++
		case models.ScanStatusPending:
			agg.pending++
		}
		if s.SpotID != "" {
			agg.spots++
		}
		if s.City != "" {
			if cityCount[phone] == nil {
				cityCount[phone] = map[string]int{}
			}
			key := s.City + "|" + s.Region + "|" + s.CountryName
			cityCount[phone][key]++
		}
	}

	for p, agg := range result {
		best := ""
		bestCount := 0
		for key, count := range cityCount[p] {
			if count > bestCount {
				best = key
				bestCount = count
			}
		}
		if best != "" {
			parts := strings.SplitN(best, "|", 3)
			if len(parts) == 3 {
				agg.usualCity = parts[0]
				agg.usualRegion = parts[1]
				agg.usualCountry = parts[2]
			}
		}
	}

	return result
}

// CustomerDetail is the full detail response for a single customer.
type CustomerDetail struct {
	PhoneNumber       string            `json:"phone_number"`
	Name              string            `json:"name"`
	ProfilePictureURL string            `json:"profile_picture_url,omitempty"`
	FirstScanAt       string            `json:"first_scan_at"`
	LastActive        string            `json:"last_active"`
	PreferredLocationID int             `json:"preferred_location_id"`
	Summary           CustomerSummary   `json:"summary"`
	UsualLocation     *LocationSummary  `json:"usual_location"`
	UsualNetwork      *NetworkSummary   `json:"usual_network"`
	Scans             []ScanDetailEntry `json:"scans"`
}

type CustomerSummary struct {
	TotalScans   int64 `json:"total_scans"`
	MatchedScans int64 `json:"matched_scans"`
	PendingScans int64 `json:"pending_scans"`
	ExpiredScans int64 `json:"expired_scans"`
	SpotsVisited int64 `json:"spots_visited"`
	RegionsSeen  int64 `json:"regions_seen"`
}

type LocationSummary struct {
	City        string `json:"city,omitempty"`
	Region      string `json:"region,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Count       int    `json:"count"`
}

type NetworkSummary struct {
	IPAddress string `json:"ip_address,omitempty"`
	ISP       string `json:"isp,omitempty"`
	NetworkOrg string `json:"network_org,omitempty"`
	Count     int    `json:"count"`
}

type ScanDetailEntry struct {
	models.Scan
	Flags ScanFlags `json:"flags"`
}

type ScanFlags struct {
	DifferentRegion bool `json:"different_region"`
	DifferentIP     bool `json:"different_ip"`
}

// GetCustomer returns full detail for a customer, including their scans with
// anomaly flags (region / IP different from their usual ones) so the admin can
// spot suspicious activity.
func GetCustomer(c *gin.Context) {
	phone := strings.TrimSpace(c.Param("phone"))
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone number is required"})
		return
	}

	var customer models.Customer
	if err := db.DB.Where("phone_number = ?", phone).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var scans []models.Scan
	if err := db.DB.
		Where("phone_number = ?", phone).
		Order("scanned_at DESC").
		Find(&scans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Derive usual location (most frequent city) and usual network (most
	// frequent IP) from the customer's scans.
	usualLoc := mostFrequentLocation(scans)
	usualNet := mostFrequentNetwork(scans)

	// Build summary + flagged entries.
	summary := CustomerSummary{}
	regions := map[string]struct{}{}
	entries := make([]ScanDetailEntry, 0, len(scans))
	for _, s := range scans {
		switch s.Status {
		case models.ScanStatusMatched:
			summary.MatchedScans++
		case models.ScanStatusPending:
			summary.PendingScans++
		case models.ScanStatusExpired:
			summary.ExpiredScans++
		}
		summary.TotalScans++
		if s.SpotID != "" {
			summary.SpotsVisited++
		}
		if s.Region != "" {
			regions[s.Region] = struct{}{}
		}

		flags := ScanFlags{}
		if usualLoc != nil && s.Region != "" && usualLoc.Region != "" &&
			s.Region != usualLoc.Region {
			flags.DifferentRegion = true
		}
		if usualNet != nil && s.IPAddress != "" && usualNet.IPAddress != "" &&
			s.IPAddress != usualNet.IPAddress {
			flags.DifferentIP = true
		}

		entries = append(entries, ScanDetailEntry{Scan: s, Flags: flags})
	}
	summary.RegionsSeen = int64(len(regions))

	c.JSON(http.StatusOK, CustomerDetail{
		PhoneNumber:         customer.PhoneNumber,
		Name:                customer.Name,
		ProfilePictureURL:   customer.ProfilePictureURL,
		FirstScanAt:         customer.FirstScanAt.Format("2006-01-02T15:04:05Z07:00"),
		LastActive:          customer.LastActive.Format("2006-01-02T15:04:05Z07:00"),
		PreferredLocationID: customer.PreferredLocationID,
		Summary:             summary,
		UsualLocation:       usualLoc,
		UsualNetwork:        usualNet,
		Scans:               entries,
	})
}

func mostFrequentLocation(scans []models.Scan) *LocationSummary {
	type key struct {
		city, region, country, code string
	}
	counts := map[key]int{}
	for _, s := range scans {
		if s.City == "" {
			continue
		}
		k := key{s.City, s.Region, s.CountryName, s.Country}
		counts[k]++
	}
	best := key{}
	bestCount := 0
	for k, count := range counts {
		if count > bestCount {
			best = k
			bestCount = count
		}
	}
	if bestCount == 0 {
		return nil
	}
	return &LocationSummary{
		City:        best.city,
		Region:      best.region,
		Country:     best.country,
		CountryCode: best.code,
		Count:       bestCount,
	}
}

func mostFrequentNetwork(scans []models.Scan) *NetworkSummary {
	type key struct {
		ip, isp, org string
	}
	counts := map[key]int{}
	for _, s := range scans {
		if s.IPAddress == "" {
			continue
		}
		k := key{s.IPAddress, s.ISP, s.NetworkOrg}
		counts[k]++
	}
	best := key{}
	bestCount := 0
	for k, count := range counts {
		if count > bestCount {
			best = k
			bestCount = count
		}
	}
	if bestCount == 0 {
		return nil
	}
	return &NetworkSummary{
		IPAddress:  best.ip,
		ISP:        best.isp,
		NetworkOrg: best.org,
		Count:      bestCount,
	}
}
