package main

import (
	"crypto/rand"
	"encoding/hex"
	"langhelper/config"
	"langhelper/database"
	"langhelper/routes"
	"log"
	"net/http"
	"sync"
)

var (
	appInstanceID string
	initOnce      sync.Once
)

func main() {
	// Генерируем уникальный ID для данного запуска приложения
	b := make([]byte, 8)
	rand.Read(b)
	appInstanceID = hex.EncodeToString(b)

	config.Init()
	database.Connect()
	router := routes.InitializeRoutes()

	router.Use(SessionCleanupMiddleware)

	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func SessionCleanupMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		initOnce.Do(func() {
			// Эта часть выполнится только один раз при первом запросе
			if cookie, err := r.Cookie("app_instance"); err != nil || cookie.Value != appInstanceID {
				// Удаляем старую сессию
				http.SetCookie(w, &http.Cookie{
					Name:     config.SessionName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				// Устанавливаем куку с ID текущего запуска приложения
				http.SetCookie(w, &http.Cookie{
					Name:     "app_instance",
					Value:    appInstanceID,
					Path:     "/",
					HttpOnly: true,
					MaxAge:   86400 * 7,
				})
				log.Println("Cleared previous session cookies")
			}
		})
		next.ServeHTTP(w, r)
	})
}
