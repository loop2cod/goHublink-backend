package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gohublink/backend/auth"
	"gohublink/backend/db"
	"gohublink/backend/models"
)

type claimTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

func ClaimScanToken(c *gin.Context) {
	var req claimTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	token := strings.ToUpper(strings.TrimSpace(req.Token))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	var scan models.Scan
	err := db.DB.Where("scan_token = ? AND status = ?", token, models.ScanStatusMatched).First(&scan).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid or unmatched token"})
		return
	}

	if scan.PhoneNumber == nil || *scan.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is not linked to a customer"})
		return
	}

	phoneNumber := *scan.PhoneNumber

	var customer models.Customer
	if err := db.DB.Where("phone_number = ?", phoneNumber).First(&customer).Error; err != nil {
		customer = models.Customer{
			PhoneNumber: phoneNumber,
			FirstScanAt: time.Now(),
			LastActive:  time.Now(),
		}
		if err := db.DB.Create(&customer).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	accessToken, err := auth.GenerateUserAccessToken(phoneNumber, auth.AccessTokenTTL())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	userRefreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	rt := models.RefreshToken{
		AdminID:     0,
		PhoneNumber: phoneNumber,
		TokenHash:   auth.HashRefreshToken(userRefreshToken),
		ExpiresAt:   time.Now().Add(auth.UserRefreshTokenTTL()),
	}
	if err := db.DB.Create(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"token":         accessToken,
		"refresh_token": userRefreshToken,
		"expires_in":    int64(auth.AccessTokenTTL().Seconds()),
		"phone_number":  phoneNumber,
	})
}

func VerifyUserToken(c *gin.Context) {
	var req claimTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	token := strings.ToUpper(strings.TrimSpace(req.Token))
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	var customer models.Customer
	if err := db.DB.Where("auth_token = ?", token).First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	phoneNumber := customer.PhoneNumber

	accessToken, err := auth.GenerateUserAccessToken(phoneNumber, auth.AccessTokenTTL())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	userRefreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	rt := models.RefreshToken{
		AdminID:     0,
		PhoneNumber: phoneNumber,
		TokenHash:   auth.HashRefreshToken(userRefreshToken),
		ExpiresAt:   time.Now().Add(auth.UserRefreshTokenTTL()),
	}
	if err := db.DB.Create(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"token":         accessToken,
		"refresh_token": userRefreshToken,
		"expires_in":    int64(auth.AccessTokenTTL().Seconds()),
		"phone_number":  phoneNumber,
	})
}

func GetCurrentUser(c *gin.Context) {
	phoneNumber, exists := c.Get("phone_number")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	phone := phoneNumber.(string)

	var customer models.Customer
	err := db.DB.Where("phone_number = ?", phone).First(&customer).Error
	if err != nil {
		customer = models.Customer{
			PhoneNumber: phone,
			FirstScanAt: time.Now(),
			LastActive:  time.Now(),
		}
		if err := db.DB.Create(&customer).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	var wallet models.Wallet
	walletErr := db.DB.Where("phone_number = ?", phone).First(&wallet).Error
	if walletErr != nil {
		wallet = models.Wallet{
			PhoneNumber: phone,
			Balance:     0,
			Tier:        WalletTierBronze,
		}
	}

	var claimedCount int64
	db.DB.Model(&models.ClaimedOffer{}).
		Where("phone_number = ? AND status = ?", phone, OfferStatusClaimed).
		Count(&claimedCount)

	c.JSON(http.StatusOK, gin.H{
		"phone_number":     customer.PhoneNumber,
		"name":             customer.Name,
		"first_scan_at":    customer.FirstScanAt,
		"last_active":      customer.LastActive,
		"wallet":           wallet,
		"claimed_offers":   claimedCount,
	})
}
