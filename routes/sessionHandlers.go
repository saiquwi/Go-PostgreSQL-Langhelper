package routes

import (
	"errors"
	"html/template"
	"log"
	"net/http"

	"langhelperCopy/config"
	"langhelperCopy/database"
	"langhelperCopy/models"

	"strings"

	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// RegisterHandler обрабатывает запросы на страницу регистрации
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Получаем данные формы
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))

		// Валидируем данные
		validationErrors := models.ValidateUser(username, password)

		if len(validationErrors) == 0 {
			db := database.GetDB()
			var existingUser models.User
			err := db.Raw("SELECT * FROM langhelpercopy.users WHERE username = ? LIMIT 1", username).Scan(&existingUser).Error

			if err == nil && existingUser.ID != 0 {
				validationErrors["username"] = errors.New("username already exists")
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("Database error: %v", err)
				validationErrors["general"] = errors.New("registration failed, please try again")
			}
		}

		// Если есть ошибки - показываем форму снова
		if len(validationErrors) > 0 {
			renderRegisterForm(w, username, validationErrors)
			return
		}

		// Создаем пользователя
		user := models.User{
			Username: username,
			Password: password, // BeforeSave хеширует пароль
		}

		db := database.GetDB()
		if err := db.Create(&user).Error; err != nil {
			log.Printf("Failed to create user: %v", err)
			validationErrors["general"] = errors.New("registration failed, please try again")
			renderRegisterForm(w, username, validationErrors)
			return
		}

		// Успешная регистрация
		http.Redirect(w, r, "/login?registered=true", http.StatusSeeOther)
		return
	}

	// Показ формы для GET-запроса
	renderRegisterForm(w, "", nil)
}

func renderRegisterForm(w http.ResponseWriter, username string, errors map[string]error) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username string
		Errors   map[string]error
	}{
		Username: username,
		Errors:   errors,
	}

	err = tmpl.ExecuteTemplate(w, "register.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

// LoginHandler обрабатывает запросы на страницу авторизации
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))

		errors := make(map[string]string)

		// Базовая валидация
		if username == "" {
			errors["Username"] = "Username is required"
		}
		if password == "" {
			errors["Password"] = "Password is required"
		}

		if len(errors) > 0 {
			renderLoginForm(w, username, errors, "")
			return
		}

		log.Printf("Login attempt for: %s", username)

		var user models.User
		db := database.GetDB()

		if err := db.Raw("SELECT * FROM langhelpercopy.users WHERE username = ? LIMIT 1", username).
			Scan(&user).Error; err != nil {
			log.Printf("User not found: %v", err)
			errors["Username"] = "Invalid username or password"
			renderLoginForm(w, username, errors, "")
			return
		}

		if err := models.ComparePassword(user.Password, password); err != nil {
			log.Printf("Invalid password for user %s", username)
			errors["Password"] = "Invalid username or password"
			renderLoginForm(w, username, errors, "")
			return
		}

		session, err := config.Store.New(r, config.SessionName)
		if err != nil {
			log.Printf("Error creating new session: %v", err)
			renderLoginForm(w, username, nil, "Internal server error. Please try again.")
			return
		}

		session.Values = map[interface{}]interface{}{
			"authenticated": true,
			"username":      username,
			"user_id":       user.ID,
		}

		if err := session.Save(r, w); err != nil {
			log.Printf("Session save error: %v", err)
			renderLoginForm(w, username, nil, "Internal server error. Please try again.")
			return
		}

		log.Printf("User %s logged in successfully", username)
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	renderLoginForm(w, "", nil, "")
}

func renderLoginForm(w http.ResponseWriter, username string, errors map[string]string, generalError string) {
	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username string
		Errors   map[string]string
		Error    string
	}{
		Username: username,
		Errors:   errors,
		Error:    generalError,
	}

	err = tmpl.ExecuteTemplate(w, "login.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

// HomeHandler обрабатывает запросы на домашнюю страницу
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Invalid session", http.StatusInternalServerError)
		return
	}

	db := database.GetDB()

	// Получаем статистику пользователя
	var deckCount int64
	row := db.Raw("SELECT COUNT(*) FROM langhelpercopy.decks WHERE user_id = ?", userID).Row()
	if err := row.Scan(&deckCount); err != nil {
		log.Printf("Failed to get deck count: %v", err)
		deckCount = 0
	}

	var cardCount int64
	err = db.Raw(`
		SELECT COUNT(dw.id) 
		FROM langhelpercopy.deck_words dw
		INNER JOIN langhelpercopy.decks d ON dw.deck_id = d.id
		WHERE d.user_id = ?
	`, userID).Scan(&cardCount).Error
	if err != nil {
		log.Printf("Failed to get card count: %v", err)
		cardCount = 0
	}

	data := struct {
		Title     string
		Username  string
		DeckCount int64
		CardCount int64
	}{
		Title:     "Home",
		Username:  session.Values["username"].(string),
		DeckCount: deckCount,
		CardCount: cardCount,
	}

	tmpl, err := template.ParseFiles("templates/layout.html", "templates/home.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

type SettingsData struct {
	Title           string
	CurrentUsername string
	NewUsername     string
	SuccessMessage  string
	ErrorMessage    string
}

func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil {
		log.Printf("Session error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Проверка аутентификации
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	currentUsername, ok := session.Values["username"].(string)
	if !ok {
		session.Values["authenticated"] = false
		session.Save(r, w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/settings/username") {
		handleUsernameChange(w, r, session, currentUsername)
		return
	}

	renderSettingsPage(w, SettingsData{
		CurrentUsername: currentUsername,
	})
}

func handleUsernameChange(w http.ResponseWriter, r *http.Request, session *sessions.Session, currentUsername string) {
	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		http.Error(w, "Invalid user session", http.StatusInternalServerError)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderSettingsPage(w, SettingsData{
			Title:           "Settings",
			CurrentUsername: currentUsername,
			ErrorMessage:    "Invalid form data",
		})
		return
	}

	data := SettingsData{
		Title:           "Settings",
		CurrentUsername: currentUsername,
		NewUsername:     strings.TrimSpace(r.FormValue("new_username")),
	}

	// Валидация
	if data.NewUsername == "" {
		data.ErrorMessage = "New username is required"
		renderSettingsPage(w, data)
		return
	}

	if len(data.NewUsername) < 3 || len(data.NewUsername) > 20 {
		data.ErrorMessage = "Username must be between 3 and 20 characters"
		renderSettingsPage(w, data)
		return
	}

	password := r.FormValue("password")
	if password == "" {
		data.ErrorMessage = "Password is required"
		renderSettingsPage(w, data)
		return
	}

	db := database.GetDB()

	// Проверка пароля
	var user models.User
	if err := db.Raw("SELECT * FROM langhelpercopy.users WHERE id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			session.Values["authenticated"] = false
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		log.Printf("Database error: %v", err)
		data.ErrorMessage = "Internal server error"
		renderSettingsPage(w, data)
		return
	}

	if err := models.ComparePassword(user.Password, password); err != nil {
		data.ErrorMessage = "Incorrect password"
		renderSettingsPage(w, data)
		return
	}

	// Проверка уникальности нового username
	var count int
	row := db.Raw("SELECT COUNT(*) FROM langhelpercopy.users WHERE username = ?", data.NewUsername).Row()
	if err := row.Scan(&count); err != nil {
		log.Printf("Database error: %v", err)
		data.ErrorMessage = "Internal server error"
		renderSettingsPage(w, data)
		return
	}

	if count > 0 {
		data.ErrorMessage = "Username already taken"
		renderSettingsPage(w, data)
		return
	}

	// Обновление username
	res := db.Exec(`
	UPDATE langhelpercopy.users 
	SET username = ? 
	WHERE id = ?`,
		data.NewUsername, user.ID)
	if res.Error != nil {
		log.Printf("Failed to update username: %v", res.Error)
		data.ErrorMessage = "Failed to update username"
		renderSettingsPage(w, data)
		return
	}

	// Обновление сессии
	session.Values["username"] = data.NewUsername
	if err := session.Save(r, w); err != nil {
		log.Printf("Failed to save session: %v", err)
		data.ErrorMessage = "Failed to update session"
		renderSettingsPage(w, data)
		return
	}

	data.CurrentUsername = data.NewUsername
	data.NewUsername = ""
	data.SuccessMessage = "Username successfully updated!"
	renderSettingsPage(w, data)
}

func renderSettingsPage(w http.ResponseWriter, data SettingsData) {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/settings.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}
