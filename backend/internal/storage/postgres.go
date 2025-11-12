package storage

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitPostgres() {
	// Connection details (you can later move these to env vars)
	host := "localhost"
	user := "postgres"
	password := "775688" // <-- replace this
	dbName := "tictactoe"
	port := "5432"

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbName, port)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}

	log.Println("✅ Connected to PostgreSQL successfully (via GORM)")

	// Auto-create tables if they don't exist
	autoMigrate()
}

func autoMigrate() {
	err := DB.AutoMigrate(&Game{}, &Score{})
	if err != nil {
		log.Fatalf("❌ Auto migration failed: %v", err)
	}
	log.Println("✅ Database schema synced successfully (AutoMigrated)")
}
