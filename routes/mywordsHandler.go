package routes

import (
	"fmt"
	"html/template"
	"langhelperCopy/config"
	"langhelperCopy/database"
	"langhelperCopy/models"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func WordsHandler(w http.ResponseWriter, r *http.Request) {
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

	var langs []models.UserLang
	if err := db.Raw("SELECT id, lang_title FROM langhelpercopy.user_langs WHERE user_id = ?", userID).Scan(&langs).Error; err != nil {
		http.Error(w, "Failed to get user langs", http.StatusInternalServerError)
		return
	}

	var formError string

	if r.Method == http.MethodPost {
		r.ParseForm()
		wordID := r.FormValue("word_id")

		var translations []struct {
			LangID      uint
			Translation string
		}
		for _, lang := range langs {
			val := strings.TrimSpace(r.FormValue(fmt.Sprintf("translation_%d", lang.ID)))
			if val != "" {
				translations = append(translations, struct {
					LangID      uint
					Translation string
				}{lang.ID, val})
			}
		}

		if len(translations) == 0 {
			formError = "At least one translation must be provided"
		} else {
			if wordID == "" {
				var newWordID uint
				err := db.Raw("INSERT INTO langhelpercopy.words DEFAULT VALUES RETURNING id").Scan(&newWordID).Error
				if err != nil {
					log.Fatalf("Failed to insert word: %v", err)
				}
				wordID = fmt.Sprint(newWordID)

				for _, t := range translations {
					db.Exec("INSERT INTO langhelpercopy.user_words (lang_id, word_id, translation) VALUES (?, ?, ?)", t.LangID, wordID, t.Translation)
				}
			} else {
				for _, t := range translations {
					var existing string
					db.Raw("SELECT translation FROM langhelpercopy.user_words WHERE word_id = ? AND lang_id = ?", wordID, t.LangID).Scan(&existing)
					if existing == "" {
						// новый перевод
						db.Exec("INSERT INTO langhelpercopy.user_words (lang_id, word_id, translation) VALUES (?, ?, ?)", t.LangID, wordID, t.Translation)
					} else if existing != t.Translation {
						// обновляем только если отличается
						db.Exec("UPDATE langhelpercopy.user_words SET translation = ? WHERE word_id = ? AND lang_id = ?", t.Translation, wordID, t.LangID)
					}
				}
			}
			if formError == "" {
				http.Redirect(w, r, "/mywords", http.StatusSeeOther)
				return
			}
		}
	}

	type WordGroup struct {
		ID           uint
		Translations []string
	}
	wordGroups := []WordGroup{}

	var wordIDs []uint
	db.Raw(`
		SELECT DISTINCT word_id FROM langhelpercopy.user_words 
		WHERE lang_id IN (SELECT id FROM langhelpercopy.user_langs WHERE user_id = ?) 
		ORDER BY word_id`, userID).Scan(&wordIDs)

	for _, wid := range wordIDs {
		trans := make([]string, len(langs))
		for i, lang := range langs {
			var t string
			db.Raw("SELECT translation FROM langhelpercopy.user_words WHERE word_id = ? AND lang_id = ?", wid, lang.ID).Scan(&t)
			trans[i] = t
		}
		wordGroups = append(wordGroups, WordGroup{
			ID:           wid,
			Translations: trans,
		})
	}

	data := map[string]interface{}{
		"Title":     "My Words",
		"Langs":     langs,
		"Words":     wordGroups,
		"FormError": formError,
	}

	tmpl, err := template.ParseFiles("templates/layout.html", "templates/mywords.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "layout.html", data)
}

func DeleteWordHandler(w http.ResponseWriter, r *http.Request) {
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

	vars := mux.Vars(r)
	wordID := vars["id"]

	db := database.GetDB()

	var count int64
	db.Raw(`
		SELECT COUNT(*) FROM langhelpercopy.user_words 
		WHERE word_id = ? AND lang_id IN (
			SELECT id FROM langhelpercopy.user_langs WHERE user_id = ?
		)
	`, wordID, userID).Scan(&count)

	if count == 0 {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Каскадное удаление связей с колодами
	db.Exec("DELETE FROM langhelpercopy.deck_words WHERE word_id = ?", wordID)

	// Удаление переводов слова
	db.Exec("DELETE FROM langhelpercopy.user_words WHERE word_id = ?", wordID)

	// Удаление самого слова
	db.Exec("DELETE FROM langhelpercopy.words WHERE id = ?", wordID)

	http.Redirect(w, r, "/mywords", http.StatusSeeOther)
}
