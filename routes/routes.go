package routes

import (
	"html/template"
	"log"
	"net/http"

	"langhelper/config"
	"langhelper/database"
	"langhelper/models"

	"errors"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// InitializeRoutes создает маршруты для приложения
func InitializeRoutes() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", IndexHandler).Methods("GET")
	router.HandleFunc("/register", RegisterHandler).Methods("GET", "POST")
	router.HandleFunc("/login", LoginHandler).Methods("GET", "POST")
	router.HandleFunc("/home", HomeHandler).Methods("GET")
	router.HandleFunc("/mywords", WordsHandler).Methods("GET", "POST")
	router.HandleFunc("/mylanguages", LanguagesHandler).Methods("GET", "POST")
	router.HandleFunc("/settings", SettingsHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", LogoutHandler).Methods("POST")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return router
}

// IndexHandler обрабатывает запросы на главную страницу
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// RegisterHandler обрабатывает запросы на страницу регистрации
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// 1. Получение и очистка данных
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))

		// 2. Базовая валидация
		if username == "" || password == "" {
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return
		}

		db := database.GetDB()

		// 3. Проверка уникальности username (опционально, т.к. gorm уже имеет unique constraint)
		var count int64
		if err := db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		user := models.User{
			Username: username,
			Password: password, // BeforeSave хеширует пароль
		}

		if err := db.Table("users").Create(&user).Error; err != nil {
			log.Printf("Failed to create user: %v", err)
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		userLangs := models.UserLangs{
			UserID: user.ID,
			Lang1:  "",
			Lang2:  "",
			Lang3:  "",
			Lang4:  "",
			Lang5:  "",
		}

		query := `INSERT INTO user_langs (user_id, lang1, lang2, lang3, lang4, lang5) VALUES (?, ?, ?, ?, ?, ?)`
		if err := db.Exec(query, userLangs.UserID, userLangs.Lang1, userLangs.Lang2, userLangs.Lang3, userLangs.Lang4, userLangs.Lang5).Error; err != nil {
			// Откатываем создание пользователя
			db.Delete(&user)
			log.Printf("Failed to create user langs: %v", err)
			http.Error(w, "Failed to initialize user settings", http.StatusInternalServerError)
			return
		}

		// 6. Успешная регистрация
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 7. Отображение формы регистрации (GET)
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// LoginHandler обрабатывает запросы на страницу авторизации
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		log.Printf("Login attempt for: %s", username)

		var user models.User
		db := database.GetDB()

		if err := db.Raw("SELECT * FROM langhelper.users WHERE username = ? LIMIT 1", username).
			Scan(&user).Error; err != nil {
			log.Printf("User not found: %v", err)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		if err := models.ComparePassword(user.Password, password); err != nil {
			log.Printf("Invalid password for user %s", username)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		session, err := config.Store.New(r, config.SessionName)
		if err != nil {
			log.Printf("Error creating new session: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		session.Values = map[interface{}]interface{}{
			"authenticated": true,
			"username":      username,
			"user_id":       user.ID,
		}

		if err := session.Save(r, w); err != nil {
			log.Printf("Session save error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("User %s logged in successfully", username)
		r.Form = nil
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// HomeHandler обрабатывает запросы на домашнюю страницу
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := session.Values["username"].(string)

	// Загрузка основного шаблона
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/home.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Отправка данных на шаблон
	err = tmpl.ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"Title":    "Home",
		"Username": username,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	// Получаем текущие языки пользователя
	var userLangs models.UserLangs

	err = db.Raw("SELECT * FROM user_langs WHERE user_id = ?", userID).Scan(&userLangs).Error

	if err != nil {
		// Проверяем, является ли ошибка "запись не найдена"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Создаем новую запись, если не существует
			result := db.Exec(`
            	INSERT INTO user_langs (user_id, lang1, lang2, lang3, lang4, lang5) 
            	VALUES (?, '', '', '', '', '')`,
				userID)

			if result.Error != nil {
				log.Printf("Failed to create user_langs record: %v", result.Error)
				http.Error(w, "Failed to initialize language settings", http.StatusInternalServerError)
				return
			}

			// Повторно запрашиваем только что созданную запись
			err = db.Raw("SELECT * FROM user_langs WHERE user_id = ?", userID).Scan(&userLangs).Error
			if err != nil {
				log.Printf("Failed to load created record: %v", err)
				http.Error(w, "Failed to load language settings", http.StatusInternalServerError)
				return
			}
		} else {
			// Если это другая ошибка
			log.Printf("Database error: %v", err)
			http.Error(w, "Failed to load language settings", http.StatusInternalServerError)
			return
		}
	}

	if r.Method == http.MethodPost {
		// Обработка сохранения формы
		r.ParseForm()

		// Обновляем языки через Raw SQL
		result := db.Exec(`
            UPDATE user_langs 
            SET lang1 = ?, lang2 = ?, lang3 = ?, lang4 = ?, lang5 = ?
            WHERE user_id = ?`,
			strings.TrimSpace(r.FormValue("lang1")),
			strings.TrimSpace(r.FormValue("lang2")),
			strings.TrimSpace(r.FormValue("lang3")),
			strings.TrimSpace(r.FormValue("lang4")),
			strings.TrimSpace(r.FormValue("lang5")),
			userID)

		if result.Error != nil {
			log.Printf("Update error: %v", err)
			http.Error(w, "Failed to update languages", http.StatusInternalServerError)
			return
		}

		// Обновляем локальную копию данных
		userLangs.Lang1 = strings.TrimSpace(r.FormValue("lang1"))
		userLangs.Lang2 = strings.TrimSpace(r.FormValue("lang2"))
		userLangs.Lang3 = strings.TrimSpace(r.FormValue("lang3"))
		userLangs.Lang4 = strings.TrimSpace(r.FormValue("lang4"))
		userLangs.Lang5 = strings.TrimSpace(r.FormValue("lang5"))

		// Перенаправляем на GET-запрос чтобы избежать повторной отправки формы
		http.Redirect(w, r, "/mylanguages", http.StatusSeeOther)
		return
	}

	// Подготавливаем данные для шаблона
	data := map[string]interface{}{
		"Title": "My Languages",
		"Langs": userLangs,
	}

	// Рендерим шаблон
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/mylanguages.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	var userLangs models.UserLangs
	err = db.Raw("SELECT lang1, lang2, lang3, lang4, lang5 FROM user_langs WHERE user_id = ?", userID).Scan(&userLangs).Error
	if err != nil {
		log.Printf("Failed to get user languages: %v", err)
		http.Error(w, "Failed to load language settings", http.StatusInternalServerError)
		return
	}

	langs := []string{userLangs.Lang1, userLangs.Lang2, userLangs.Lang3, userLangs.Lang4, userLangs.Lang5}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		translations := r.Form["translations[]"]

		// Проверяем, что хотя бы один перевод не пустой
		atLeastOneFilled := false
		for _, trans := range translations {
			if strings.TrimSpace(trans) != "" {
				atLeastOneFilled = true
				break
			}
		}

		if !atLeastOneFilled {
			http.Error(w, "At least one translation must be provided", http.StatusBadRequest)
			return
		}

		// Создаем новую запись слова
		res := db.Exec(`
			INSERT INTO user_words 
			(user_id, tran1, tran2, tran3, tran4, tran5) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			userID,
			safeGetTranslation(translations, 0),
			safeGetTranslation(translations, 1),
			safeGetTranslation(translations, 2),
			safeGetTranslation(translations, 3),
			safeGetTranslation(translations, 4))

		if res.Error != nil {
			log.Printf("Failed to insert word: %v", res.Error)
			http.Error(w, "Failed to save word", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/mywords", http.StatusSeeOther)
		return
	}

	var words []models.UserWords
	err = db.Raw("SELECT id, tran1, tran2, tran3, tran4, tran5 FROM user_words WHERE user_id = ?", userID).Scan(&words).Error
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Failed to load words", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "My Words",
		"Words": words,
		"Langs": langs,
	}

	tmpl, err := template.ParseFiles("templates/layout.html", "templates/mywords.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func safeGetTranslation(translations []string, index int) string {
	// Проверяем, существует ли элемент с указанным индексом в срезе
	if index < len(translations) {
		// Если элемент существует, возвращаем его, предварительно обрезав пробелы
		return strings.TrimSpace(translations[index])
	}
	// Если элемента с таким индексом нет, возвращаем пустую строку
	return ""
}

func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Загрузка основного шаблона
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/settings.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Отправка данных на шаблон
	err = tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title": "Settings",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// LogoutHandler обрабатывает запросы на выход
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Полная очистка сессии
	session.Options.MaxAge = -1 // Удалить cookie
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to destroy session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
