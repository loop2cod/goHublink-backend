package db

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() error {
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		dsn = buildDSN()
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("unable to get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(0)

	DB = gormDB
	log.Println("Database connection established")
	return nil
}

func Close() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
			log.Println("Database connection closed")
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func buildDSN() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := getEnv("DB_SSLMODE", "disable")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("Database configuration incomplete: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, and DB_NAME must be set")
	}

	if isRailwayEnvironment() && !strings.Contains(host, "railway.internal") {
		privateDomain := os.Getenv("RAILWAY_PRIVATE_DOMAIN")
		if privateDomain != "" && !strings.Contains(host, ".") {
			host = "postgresql." + privateDomain
		}
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode,
	)
}

func isRailwayEnvironment() bool {
	return os.Getenv("RAILWAY_ENVIRONMENT") != "" || os.Getenv("RAILWAY_PRIVATE_DOMAIN") != ""
}