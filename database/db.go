package database

import (
	"log"

	"langhelper/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

// Connect создает подключение к базе данных
func Connect() {
	var err error
	dsn := "host=localhost user=langhelper password=langhelper123 dbname=postgres port=5432 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	// Установка опций для таблиц
	db = db.Set("gorm:table_options", "WITH (OIDS=FALSE)")

	// Установка схемы по умолчанию
	if err := db.Exec("SET search_path TO langhelper").Error; err != nil {
		log.Fatal("failed to set search path:", err)
	}

	// Автоматическая миграция для создания таблицы User
	if err := db.AutoMigrate(&models.User{}, &models.UserLangs{}, &models.UserWords{}); err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("Connected to database and migrated!")
}

// GetDB возвращает текущее соединение с базой данных
func GetDB() *gorm.DB {
	return db
}
