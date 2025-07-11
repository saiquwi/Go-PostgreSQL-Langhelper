package routes

import (
	"html/template"
	"langhelperCopy/config"
	"langhelperCopy/database"
	"langhelperCopy/models"
	"net/http"
	"strconv"
)

type LangPageData struct {
	Title     string
	Languages []models.UserLang
	EditID    int
}

func LanguagesHandler(w http.ResponseWriter, r *http.Request) {
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

	editIDParam := r.URL.Query().Get("edit")
	editID := 0
	if editIDParam != "" {
		tmp, _ := strconv.Atoi(editIDParam)
		editID = tmp
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		langtitle := r.FormValue("langtitle")
		if langtitle != "" {
			result := db.Exec("INSERT INTO langhelpercopy.user_langs (user_id, lang_title) VALUES (?, ?)", userID, langtitle)
			if result.Error != nil {
				http.Error(w, "Error inserting language", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/mylanguages", http.StatusSeeOther)
			return
		}
	}

	rows, err := db.Raw("SELECT id, user_id, lang_title FROM langhelpercopy.user_langs WHERE user_id = ?", userID).Rows()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var languages []models.UserLang
	for rows.Next() {
		var lang models.UserLang
		err := rows.Scan(&lang.ID, &lang.UserID, &lang.LangTitle)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		languages = append(languages, lang)
	}

	tmpl, _ := template.ParseFiles("templates/layout.html", "templates/mylanguages.html")
	tmpl.ExecuteTemplate(w, "layout.html", LangPageData{
		Title:     "My Languages",
		Languages: languages,
		EditID:    editID,
	})
}

func EditLanguageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/mylanguages", http.StatusSeeOther)
		return
	}

	db := database.GetDB()

	idStr := r.URL.Path[len("/mylanguages/edit/"):]
	id, _ := strconv.Atoi(idStr)
	newTitle := r.FormValue("newtitle")

	result := db.Exec("UPDATE langhelpercopy.user_langs SET lang_title = ? WHERE id = ?", newTitle, id)
	if result.Error != nil {
		http.Error(w, "Error updating language", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/mylanguages", http.StatusSeeOther)
}

func DeleteLanguageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/mylanguages", http.StatusSeeOther)
		return
	}

	db := database.GetDB()

	idStr := r.URL.Path[len("/mylanguages/delete/"):]
	id, _ := strconv.Atoi(idStr)

	_ = db.Exec("DELETE FROM langhelpercopy.user_words WHERE lang_id = ?", id)
	_ = db.Exec("DELETE FROM langhelpercopy.deck_langs WHERE lang_id = ?", id)

	result := db.Exec("DELETE FROM langhelpercopy.user_langs WHERE id = ?", id)
	if result.Error != nil {
		http.Error(w, "Error deleting language", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/mylanguages", http.StatusSeeOther)
}
