package routes

import (
	"fmt"
	"html/template"
	"langhelperCopy/config"
	"langhelperCopy/database"
	"langhelperCopy/models"
	"net/http"
	"strings"
)

func DecksHandler(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()

	// Сессия и аутентификация
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	auth, ok := session.Values["authenticated"].(bool)
	if !ok || !auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Authentication error", http.StatusUnauthorized)
		return
	}

	// Получаем языки пользователя
	var userLangs []models.UserLang
	if err := db.Raw("SELECT * FROM langhelpercopy.user_langs WHERE user_id = ?", userID).Scan(&userLangs).Error; err != nil {
		http.Error(w, "Failed to fetch user languages", http.StatusInternalServerError)
		return
	}
	fmt.Printf("Loaded userLangs: %+v\n", userLangs)

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		deckTitle := strings.TrimSpace(r.FormValue("deck_title"))

		if deckTitle == "" {
			http.Error(w, "Deck title is required", http.StatusBadRequest)
			return
		}

		// Создание колоды
		var deckID uint
		err := db.Raw(
			"INSERT INTO langhelpercopy.decks (user_id, deck_title) VALUES (?, ?) RETURNING id",
			userID, deckTitle,
		).Scan(&deckID).Error
		if err != nil {
			http.Error(w, "Failed to create deck", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/mydecks", http.StatusSeeOther)
		return
	}

	// Загружаем все колоды пользователя
	var decks []models.Deck
	if err := db.Raw("SELECT * FROM langhelpercopy.decks WHERE user_id = ?", userID).Scan(&decks).Error; err != nil {
		http.Error(w, "Failed to load decks", http.StatusInternalServerError)
		return
	}

	// Для каждой колоды загружаем её языки
	type DeckWithLangs struct {
		models.Deck
		Languages []string
	}
	var decksWithLangs []DeckWithLangs
	for _, d := range decks {
		var langTitles []string
		db.Raw(`
			SELECT ul.lang_title
			FROM langhelpercopy.deck_langs dl
			JOIN langhelpercopy.user_langs ul ON dl.lang_id = ul.id
			WHERE dl.deck_id = ?`, d.ID).Scan(&langTitles)

		decksWithLangs = append(decksWithLangs, DeckWithLangs{
			Deck:      d,
			Languages: langTitles,
		})
	}

	data := map[string]interface{}{
		"Title":     "My Decks",
		"Decks":     decksWithLangs,
		"UserLangs": userLangs,
	}

	tmpl := template.New("layout.html").Funcs(template.FuncMap{
		"join": strings.Join,
	})
	tmpl, err = tmpl.ParseFiles("templates/layout.html", "templates/mydecks.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "Render error", http.StatusInternalServerError)
	}
}
