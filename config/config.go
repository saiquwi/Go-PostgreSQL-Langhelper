package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/sessions"
)

var (
	Store       *sessions.CookieStore
	storeOnce   sync.Once
	SessionName = "langhelperCopy-session" // Сделал переменной для гибкости
)

func Init() {
	storeOnce.Do(func() {
		authKey := mustGetKey("SESSION_AUTH_KEY", 32)
		encKey := mustGetKey("SESSION_ENC_KEY", 32)

		Store = sessions.NewCookieStore(authKey, encKey)

		Store.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7, // 1 неделя
			HttpOnly: true,
			Secure:   false, // В production должно быть true
			SameSite: http.SameSiteLaxMode,
		}
	})
}

func mustGetKey(envVar string, length int) []byte {
	if key := os.Getenv(envVar); key != "" {
		if len(key) < length {
			log.Fatalf("Key %s too short: need %d bytes, got %d", envVar, length, len(key))
		}
		return []byte(key)[:length]
	}

	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate random key: %v", err)
	}
	log.Printf("Generated new key for %s: %s", envVar, hex.EncodeToString(key))
	return key
}
