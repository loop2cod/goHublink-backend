package db

import (
	"log"

	"golang.org/x/crypto/bcrypt"

	"gohublink/backend/models"
)

// Migrate syncs the database schema with the models.
// Any change made to a model struct is applied automatically on server start.
func Migrate() error {
	if err := DB.AutoMigrate(&models.Admin{}, &models.Spot{}, &models.RefreshToken{}); err != nil {
		return err
	}
	log.Println("Database schema synced with models")
	return nil
}

func Seed() error {
	var adminCount int64
	if err := DB.Model(&models.Admin{}).Count(&adminCount).Error; err != nil {
		return err
	}
	if adminCount == 0 {
		username := getEnv("ADMIN_USERNAME", "admin")
		password := getEnv("ADMIN_PASSWORD", "admin123")
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := DB.Create(&models.Admin{
			Username: username,
			Password: string(hash),
		}).Error; err != nil {
			return err
		}
		log.Printf("Seeded default admin user %q", username)
	}

	var spotCount int64
	if err := DB.Model(&models.Spot{}).Where("id = ?", "ONLINE").Count(&spotCount).Error; err != nil {
		return err
	}
	if spotCount == 0 {
		if err := DB.Create(&models.Spot{
			ID:        "ONLINE",
			Name:      "Online",
			Latitude:  0,
			Longitude: 0,
		}).Error; err != nil {
			return err
		}
		log.Println("Seeded default spot with id ONLINE")
	}
	return nil
}