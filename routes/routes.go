package routes

import (
	"html/template"
	"langhelperCopy/config"
	"net/http"

	"github.com/gorilla/mux"
)

func InitializeRoutes() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler).Methods("GET")
	router.HandleFunc("/register", RegisterHandler).Methods("GET", "POST")
	router.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	router.HandleFunc("/home", HomeHandler).Methods("GET")
	router.HandleFunc("/settings", SettingsHandler).Methods("GET", "POST")
	router.HandleFunc("/settings/username", SettingsHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", LogoutHandler).Methods("GET", "POST")

	router.HandleFunc("/mylanguages", LanguagesHandler).Methods("GET", "POST")
	router.HandleFunc("/mylanguages/edit/{id:[0-9]+}", EditLanguageHandler).Methods("GET", "POST")
	router.HandleFunc("/mylanguages/delete/{id:[0-9]+}", DeleteLanguageHandler).Methods("GET", "POST")
	router.HandleFunc("/mywords", WordsHandler).Methods("GET", "POST")
	router.HandleFunc("/mywords/delete/{id}", DeleteWordHandler).Methods("GET", "POST")

	router.HandleFunc("/mydecks", DecksHandler).Methods("GET", "POST")
	router.HandleFunc("/deck/{id:[0-9]+}", ViewDeckHandler).Methods("GET", "POST")
	router.HandleFunc("/deck/addlang/{id:[0-9]+}", AddLangToDeckHandler).Methods("POST")
	router.HandleFunc("/deck/removelang/{deck_id:[0-9]+}/{lang_id:[0-9]+}", RemoveLangFromDeckHandler).Methods("POST")
	router.HandleFunc("/decks/addword", AddWordToDeckHandler).Methods("POST")
	router.HandleFunc("/decks/removeword", RemoveWordFromDeckHandler).Methods("POST")

	router.HandleFunc("/flashcards", FlashcardsHandler).Methods("GET", "POST")
	router.HandleFunc("/flashcards/check", FlashcardsCheckHandler).Methods("POST")

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return router
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to destroy session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
