package database

import (
	"log"

	"langhelperCopy/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func Connect() {
	var err error
	dsn := "host=localhost user=langhelper password=langhelper123 dbname=postgres port=5432 sslmode=disable search_path=langhelpercopy"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	db = db.Set("gorm:table_options", "WITH (OIDS=FALSE)")

	if err := db.AutoMigrate(&models.User{}, &models.UserLang{}, &models.UserWord{}, &models.Deck{}, &models.DeckWord{}, &models.DeckLang{}); err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("Connected to database and migrated!")
}

func GetDB() *gorm.DB {
	return db
}
