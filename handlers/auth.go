package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"gohublink/backend/auth"
	"gohublink/backend/db"
	"gohublink/backend/models"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func issueTokens(c *gin.Context, admin *models.Admin) {
	access, err := auth.GenerateAccessToken(admin.Username, auth.AccessTokenTTL())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refresh, err := auth.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	rt := models.RefreshToken{
		AdminID:   admin.ID,
		TokenHash: auth.HashRefreshToken(refresh),
		ExpiresAt: time.Now().Add(auth.RefreshTokenTTL()),
	}
	if err := db.DB.Create(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"token":         access, // backwards compatible
		"refresh_token": refresh,
		"expires_in":    int64(auth.AccessTokenTTL().Seconds()),
	})
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	if err := db.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	issueTokens(c, &admin)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	hash := auth.HashRefreshToken(req.RefreshToken)

	var stored models.RefreshToken
	if err := db.DB.Where("token_hash = ?", hash).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired or revoked"})
		return
	}

	// Rotate: revoke this token since it's being consumed (reuse detection).
	now := time.Now()
	if err := db.DB.Model(&stored).Update("revoked_at", now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var admin models.Admin
	if err := db.DB.First(&admin, stored.AdminID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	issueTokens(c, &admin)
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	hash := auth.HashRefreshToken(req.RefreshToken)
	now := time.Now()

	result := db.DB.Model(&models.RefreshToken{}).
		Where("token_hash = ?", hash).
		Update("revoked_at", now)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

// RevokeAllUserTokens revokes every refresh token belonging to a user id.
func RevokeAllUserTokens(adminID uint) error {
	return db.DB.Model(&models.RefreshToken{}).
		Where("admin_id = ? AND revoked_at IS NULL", adminID).
		Updates(map[string]interface{}{"revoked_at": time.Now()}).
		Error
}