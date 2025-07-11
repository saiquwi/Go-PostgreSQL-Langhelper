package routes

import (
	"html/template"
	"langhelperCopy/config"
	"langhelperCopy/database"
	"langhelperCopy/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func ViewDeckHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Invalid user session", http.StatusInternalServerError)
		return
	}

	db := database.GetDB()
	vars := mux.Vars(r)
	deckIDStr := vars["id"]
	deckID, err := strconv.Atoi(deckIDStr)
	if err != nil {
		http.Error(w, "Invalid deck ID", http.StatusBadRequest)
		return
	}

	// Получение самой колоды
	var deck models.Deck
	if err := db.Raw(`SELECT * FROM langhelpercopy.decks WHERE id = ? AND user_id = ?`, deckID, userID).Scan(&deck).Error; err != nil {
		http.Error(w, "Deck not found", http.StatusNotFound)
		return
	}

	// Получение языков колоды (DeckLangs)
	var deckLangs []models.UserLang
	err = db.Raw(`
		SELECT l.id, l.lang_title, l.user_id
		FROM langhelpercopy.deck_langs dl 
		JOIN langhelpercopy.user_langs l ON dl.lang_id = l.id 
		WHERE dl.deck_id = ?
		ORDER BY l.lang_title
	`, deckID).Scan(&deckLangs).Error
	if err != nil {
		http.Error(w, "Failed to load deck languages", http.StatusInternalServerError)
		return
	}

	// Языки пользователя, которые ещё не добавлены в колоду (AvailableLangs)
	var availableLangs []models.UserLang
	err = db.Raw(`
		SELECT l.id, l.lang_title, l.user_id
		FROM langhelpercopy.user_langs l
		WHERE l.user_id = ? AND l.id NOT IN (
			SELECT lang_id FROM langhelpercopy.deck_langs WHERE deck_id = ?
		)
		ORDER BY l.lang_title
	`, userID, deckID).Scan(&availableLangs).Error
	if err != nil {
		http.Error(w, "Failed to load available languages", http.StatusInternalServerError)
		return
	}

	// Получение word_id из deck_words
	var wordIDsInDeck []int
	err = db.Raw(`SELECT word_id FROM langhelpercopy.deck_words WHERE deck_id = ?`, deckID).Scan(&wordIDsInDeck).Error
	if err != nil {
		http.Error(w, "Failed to load deck words", http.StatusInternalServerError)
		return
	}

	// Получение переводов для этих слов только по языкам колоды
	type WordWithTranslations struct {
		WordID       int
		Translations map[string]string
	}

	deckWords := make([]WordWithTranslations, 0)

	if len(wordIDsInDeck) > 0 {
		langIDs := make([]int, len(deckLangs))
		for i, dl := range deckLangs {
			langIDs[i] = int(dl.ID)
		}

		rows, err := db.Raw(`
			SELECT uw.word_id, uw.translation, ul.lang_title
			FROM langhelpercopy.user_words uw
			JOIN langhelpercopy.user_langs ul ON uw.lang_id = ul.id
			WHERE uw.word_id IN (?) AND uw.lang_id IN (?)
		`, wordIDsInDeck, langIDs).Rows()
		if err != nil {
			http.Error(w, "Failed to load word translations", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		wordMap := make(map[int]map[string]string)
		for rows.Next() {
			var wordID int
			var translation, langTitle string
			if err := rows.Scan(&wordID, &translation, &langTitle); err != nil {
				http.Error(w, "Error scanning rows", http.StatusInternalServerError)
				return
			}
			if _, ok := wordMap[wordID]; !ok {
				wordMap[wordID] = make(map[string]string)
			}
			wordMap[wordID][langTitle] = translation
		}

		for id, translations := range wordMap {
			deckWords = append(deckWords, WordWithTranslations{
				WordID:       id,
				Translations: translations,
			})
		}
	}

	// Получение candidate word_ids (у пользователя, но не в колоде)
	var candidateWordIDs []int
	err = db.Raw(`
		SELECT DISTINCT uw.word_id
		FROM langhelpercopy.user_words uw
		JOIN langhelpercopy.user_langs ul ON uw.lang_id = ul.id
		WHERE ul.user_id = ?
		AND uw.word_id NOT IN (
			SELECT word_id FROM langhelpercopy.deck_words WHERE deck_id = ?
		)
	`, userID, deckID).Scan(&candidateWordIDs).Error
	if err != nil {
		http.Error(w, "Failed to load candidate words", http.StatusInternalServerError)
		return
	}

	// Отбор слов, у которых есть переводы на все языки колоды
	availableWords := make([]WordWithTranslations, 0)

	if len(candidateWordIDs) > 0 && len(deckLangs) > 0 {
		langIDs := make([]int, len(deckLangs))
		for i, l := range deckLangs {
			langIDs[i] = int(l.ID)
		}

		rows, err := db.Raw(`
			SELECT uw.word_id, uw.translation, ul.lang_title
			FROM langhelpercopy.user_words uw
			JOIN langhelpercopy.user_langs ul ON uw.lang_id = ul.id
			WHERE ul.user_id = ?
			AND uw.word_id IN (?)
			AND uw.lang_id IN (?)
			ORDER BY uw.word_id
		`, userID, candidateWordIDs, langIDs).Rows()
		if err != nil {
			http.Error(w, "Failed to load available word translations", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		wordMap := make(map[int]map[string]string)
		counts := make(map[int]int)

		for rows.Next() {
			var wordID int
			var translation, langTitle string
			if err := rows.Scan(&wordID, &translation, &langTitle); err != nil {
				http.Error(w, "Error scanning rows", http.StatusInternalServerError)
				return
			}
			if _, ok := wordMap[wordID]; !ok {
				wordMap[wordID] = make(map[string]string)
			}
			wordMap[wordID][langTitle] = translation
			counts[wordID]++
		}

		for id, translations := range wordMap {
			if counts[id] == len(deckLangs) {
				availableWords = append(availableWords, WordWithTranslations{
					WordID:       id,
					Translations: translations,
				})
			}
		}
	}

	data := struct {
		Title              string
		Deck               models.Deck
		DeckLanguages      []models.UserLang
		DeckWords          []WordWithTranslations
		AvailableLanguages []models.UserLang
		AvailableWords     []WordWithTranslations
	}{
		Title:              "View deck",
		Deck:               deck,
		DeckLanguages:      deckLangs,
		DeckWords:          deckWords,
		AvailableLanguages: availableLangs,
		AvailableWords:     availableWords,
	}

	tmpl, err := template.ParseFiles("templates/layout.html", "templates/viewDeck.html")
	if err != nil {
		log.Printf("template.ParseFiles error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("tmpl.ExecuteTemplate error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

}

func AddLangToDeckHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Invalid user session", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	deckIDStr := vars["id"]
	deckID, err := strconv.Atoi(deckIDStr)
	if err != nil {
		http.Error(w, "Invalid deck ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	langID, err := strconv.Atoi(r.FormValue("lang_id"))
	if err != nil {
		http.Error(w, "Invalid language ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Проверка, что язык принадлежит пользователю
	var count int
	row := db.Raw("SELECT COUNT(*) FROM langhelpercopy.user_langs WHERE id = ? AND user_id = ?", langID, userID).Row()
	row.Scan(&count)
	if count == 0 {
		http.Error(w, "Unauthorized language", http.StatusForbidden)
		return
	}

	db.Exec("INSERT INTO langhelpercopy.deck_langs (deck_id, lang_id) VALUES (?, ?)", deckID, langID)
	http.Redirect(w, r, "/deck/"+strconv.Itoa(deckID), http.StatusSeeOther)
}

func RemoveLangFromDeckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	deckIDStr := vars["deck_id"]
	langIDStr := vars["lang_id"]

	deckID, err1 := strconv.Atoi(deckIDStr)
	langID, err2 := strconv.Atoi(langIDStr)
	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	db.Exec("DELETE FROM langhelpercopy.deck_langs WHERE deck_id = ? AND lang_id = ?", deckID, langID)

	http.Redirect(w, r, "/deck/"+strconv.Itoa(deckID), http.StatusSeeOther)
}

func AddWordToDeckHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Invalid user session", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	deckIDStr := r.FormValue("deck_id")
	wordIDStr := r.FormValue("word_id")

	deckID, err := strconv.Atoi(deckIDStr)
	if err != nil {
		http.Error(w, "Invalid deck ID", http.StatusBadRequest)
		return
	}

	wordID, err := strconv.Atoi(wordIDStr)
	if err != nil {
		http.Error(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Проверяем, что колода принадлежит текущему пользователю
	var count int64
	err = db.Raw(`SELECT COUNT(*) FROM langhelpercopy.decks WHERE id = ? AND user_id = ?`, deckID, userID).Scan(&count).Error
	if err != nil || count == 0 {
		http.Error(w, "Deck not found or access denied", http.StatusForbidden)
		return
	}

	// Проверяем, что слово существует у пользователя (в user_words)
	err = db.Raw(`SELECT COUNT(DISTINCT word_id) FROM langhelpercopy.user_words WHERE word_id = ? AND lang_id IN (SELECT id FROM langhelpercopy.user_langs WHERE user_id = ?)`, wordID, userID).Scan(&count).Error
	if err != nil || count == 0 {
		http.Error(w, "Word not found or access denied", http.StatusForbidden)
		return
	}

	// Проверяем, что слово еще не в колоде
	err = db.Raw(`SELECT COUNT(*) FROM langhelpercopy.deck_words WHERE deck_id = ? AND word_id = ?`, deckID, wordID).Scan(&count).Error
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		// Уже есть — можно просто редиректнуть без ошибки
		http.Redirect(w, r, "/deck/view/"+deckIDStr, http.StatusSeeOther)
		return
	}

	// Добавляем слово в колоду
	err = db.Exec(`INSERT INTO langhelpercopy.deck_words (deck_id, word_id) VALUES (?, ?)`, deckID, wordID).Error
	if err != nil {
		http.Error(w, "Failed to add word to deck", http.StatusInternalServerError)
		return
	}

	// После добавления редирект на страницу колоды
	http.Redirect(w, r, "/deck/"+deckIDStr, http.StatusSeeOther)
}

func RemoveWordFromDeckHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Invalid user session", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	deckIDStr := r.FormValue("deck_id")
	wordIDStr := r.FormValue("word_id")

	deckID, err := strconv.Atoi(deckIDStr)
	if err != nil {
		http.Error(w, "Invalid deck ID", http.StatusBadRequest)
		return
	}

	wordID, err := strconv.Atoi(wordIDStr)
	if err != nil {
		http.Error(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Проверяем, что колода принадлежит текущему пользователю
	var count int64
	err = db.Raw(`SELECT COUNT(*) FROM langhelpercopy.decks WHERE id = ? AND user_id = ?`, deckID, userID).Scan(&count).Error
	if err != nil || count == 0 {
		http.Error(w, "Deck not found or access denied", http.StatusForbidden)
		return
	}

	// Удаляем слово из колоды
	err = db.Exec(`DELETE FROM langhelpercopy.deck_words WHERE deck_id = ? AND word_id = ?`, deckID, wordID).Error
	if err != nil {
		http.Error(w, "Failed to remove word from deck", http.StatusInternalServerError)
		return
	}

	// Редирект обратно на страницу колоды
	http.Redirect(w, r, "/deck/"+deckIDStr, http.StatusSeeOther)
}
