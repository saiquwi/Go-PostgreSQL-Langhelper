package routes

import (
	"fmt"
	"html/template"
	"langhelperCopy/config"
	"langhelperCopy/database"
	"langhelperCopy/models"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
)

type LangTest struct {
	DeckLang models.DeckLang
	Options  []string
	Correct  string
}

type WordTest struct {
	WordID   uint
	MainWord string
	Tests    []LangTest
}

func FlashcardsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		log.Printf("Unauthorized access or session error: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(uint)
	if !ok {
		log.Printf("Invalid user session: cannot get user_id")
		http.Error(w, "Invalid user session", http.StatusInternalServerError)
		return
	}

	db := database.GetDB()

	// Загружаем колоды пользователя
	var decks []models.Deck
	err = db.Raw("SELECT * FROM langhelpercopy.decks WHERE user_id = ? ORDER BY deck_title", userID).Scan(&decks).Error
	if err != nil {
		log.Printf("Failed to load decks: %v", err)
		http.Error(w, "Failed to load decks", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/layout.html", "templates/flashcards.html")
		if err != nil {
			log.Printf("template.ParseFiles error on GET: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := struct {
			Title     string
			Decks     []models.Deck
			Deck      *models.Deck
			DeckLangs []models.DeckLang
			MainLang  uint
			WordTests []WordTest
		}{
			Title: "Flashcards",
			Decks: decks,
		}

		err = tmpl.ExecuteTemplate(w, "layout.html", data)
		if err != nil {
			log.Printf("ExecuteTemplate error on GET: %v", err)
		}
		return
	}

	// POST обработка
	if err := r.ParseForm(); err != nil {
		log.Printf("ParseForm error: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	step := r.FormValue("step")
	switch step {
	case "select_deck":
		handleSelectDeck(w, r, userID, decks)
	case "select_lang":
		handleSelectLang(w, r, userID, decks)
	default:
		http.Error(w, "Invalid step", http.StatusBadRequest)
	}
}

func handleSelectDeck(w http.ResponseWriter, r *http.Request, userID uint, decks []models.Deck) {
	deckID, err := strconv.ParseUint(r.FormValue("deck_id"), 10, 64)
	if err != nil {
		log.Printf("Invalid deck ID: %v", err)
		http.Error(w, "Invalid deck ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	var deck models.Deck
	err = db.Raw("SELECT * FROM langhelpercopy.decks WHERE id = ?", deckID).Scan(&deck).Error
	if err != nil {
		log.Printf("Failed to find deck: %v", err)
		http.Error(w, "Deck not found", http.StatusNotFound)
		return
	}

	var deckLangs []models.DeckLang
	err = db.Raw("SELECT * FROM langhelpercopy.deck_langs WHERE deck_id = ?", deckID).Scan(&deckLangs).Error
	if err != nil {
		log.Printf("Failed to load deck languages: %v", err)
		http.Error(w, "Failed to load deck languages", http.StatusInternalServerError)
		return
	}

	// Загружаем названия языков
	for i := range deckLangs {
		var userLang models.UserLang
		err = db.Raw("SELECT * FROM langhelpercopy.user_langs WHERE id = ?", deckLangs[i].LangID).Scan(&userLang).Error
		if err != nil {
			log.Printf("Failed to load language title: %v", err)
			continue
		}
		deckLangs[i].UserLang = userLang
	}

	tmpl, err := template.ParseFiles("templates/layout.html", "templates/flashcards.html")
	if err != nil {
		log.Printf("template.ParseFiles error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title     string
		Decks     []models.Deck
		Deck      *models.Deck
		DeckLangs []models.DeckLang
		MainLang  uint
		WordTests []WordTest
	}{
		Title:     "Flashcards",
		Decks:     decks,
		Deck:      &deck,
		DeckLangs: deckLangs,
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ExecuteTemplate error: %v", err)
	}
}

func handleSelectLang(w http.ResponseWriter, r *http.Request, userID uint, decks []models.Deck) {
	deckID, err := strconv.ParseUint(r.FormValue("deck_id"), 10, 64)
	if err != nil {
		log.Printf("Invalid deck ID: %v", err)
		http.Error(w, "Invalid deck ID", http.StatusBadRequest)
		return
	}

	mainLangID, err := strconv.ParseUint(r.FormValue("main_lang_id"), 10, 64)
	if err != nil {
		log.Printf("Invalid main language ID: %v", err)
		http.Error(w, "Invalid main language ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Загружаем колоду
	var deck models.Deck
	err = db.Raw("SELECT * FROM langhelpercopy.decks WHERE id = ?", deckID).Scan(&deck).Error
	if err != nil {
		log.Printf("Failed to find deck: %v", err)
		http.Error(w, "Deck not found", http.StatusNotFound)
		return
	}

	// Загружаем языки колоды
	var deckLangs []models.DeckLang
	err = db.Raw("SELECT * FROM langhelpercopy.deck_langs WHERE deck_id = ?", deckID).Scan(&deckLangs).Error
	if err != nil {
		log.Printf("Failed to load deck languages: %v", err)
		http.Error(w, "Failed to load deck languages", http.StatusInternalServerError)
		return
	}

	// Загружаем слова из колоды
	var deckWords []models.DeckWord
	err = db.Raw("SELECT * FROM langhelpercopy.deck_words WHERE deck_id = ?", deckID).Scan(&deckWords).Error
	if err != nil {
		log.Printf("Failed to load words: %v", err)
		http.Error(w, "Failed to load words", http.StatusInternalServerError)
		return
	}

	if len(deckWords) == 0 {
		http.Error(w, "No words in deck", http.StatusBadRequest)
		return
	}

	// Собираем ID слов
	wordIDs := make([]uint, len(deckWords))
	for i, dw := range deckWords {
		wordIDs[i] = dw.WordID
	}

	// Формируем IN условие для SQL запроса
	inClause := "("
	for i, id := range wordIDs {
		if i > 0 {
			inClause += ","
		}
		inClause += strconv.FormatUint(uint64(id), 10)
	}
	inClause += ")"

	// Загружаем основные переводы
	var mainTranslations []models.UserWord
	err = db.Raw("SELECT * FROM langhelpercopy.user_words WHERE word_id IN "+inClause+" AND lang_id = ?", mainLangID).Scan(&mainTranslations).Error
	if err != nil {
		log.Printf("Failed to load main translations: %v", err)
		http.Error(w, "Failed to load main translations", http.StatusInternalServerError)
		return
	}

	mainMap := make(map[uint]string)
	for _, uw := range mainTranslations {
		mainMap[uw.WordID] = uw.Translation
	}

	// Формируем тесты
	var wordTests []WordTest
	for _, wid := range wordIDs {
		mainWord, ok := mainMap[wid]
		if !ok {
			continue
		}

		wt := WordTest{
			WordID:   wid,
			MainWord: mainWord,
		}

		// Для каждого языка (кроме основного) создаем тест
		for _, dl := range deckLangs {
			if dl.LangID == uint(mainLangID) {
				continue
			}

			// Загружаем правильный перевод
			var correct models.UserWord
			err = db.Raw("SELECT * FROM langhelpercopy.user_words WHERE word_id = ? AND lang_id = ? LIMIT 1", wid, dl.LangID).Scan(&correct).Error
			if err != nil {
				continue
			}

			// Загружаем 4 случайных неправильных варианта
			var wrongOptions []models.UserWord
			err = db.Raw("SELECT * FROM langhelpercopy.user_words WHERE lang_id = ? AND translation != ? ORDER BY RANDOM() LIMIT 4", dl.LangID, correct.Translation).Scan(&wrongOptions).Error
			if err != nil {
				log.Printf("Failed to load wrong options: %v", err)
				continue
			}

			// Формируем варианты ответов
			options := make([]string, 0, 5)
			options = append(options, correct.Translation)
			for _, wo := range wrongOptions {
				options = append(options, wo.Translation)
			}

			// Перемешиваем варианты
			rand.Shuffle(len(options), func(i, j int) {
				options[i], options[j] = options[j], options[i]
			})

			// Загружаем информацию о языке
			var userLang models.UserLang
			err = db.Raw("SELECT * FROM langhelpercopy.user_langs WHERE id = ?", dl.LangID).Scan(&userLang).Error
			if err != nil {
				continue
			}
			dl.UserLang = userLang

			wt.Tests = append(wt.Tests, LangTest{
				DeckLang: dl,
				Options:  options,
				Correct:  correct.Translation,
			})
		}

		if len(wt.Tests) > 0 {
			wordTests = append(wordTests, wt)
		}
	}

	tmpl, err := template.ParseFiles("templates/layout.html", "templates/flashcards.html")
	if err != nil {
		log.Printf("template.ParseFiles error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title     string
		Decks     []models.Deck
		Deck      *models.Deck
		DeckLangs []models.DeckLang
		MainLang  uint
		WordTests []WordTest
	}{
		Title:     "Flashcards",
		Decks:     decks,
		Deck:      &deck,
		DeckLangs: deckLangs,
		MainLang:  uint(mainLangID),
		WordTests: wordTests,
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ExecuteTemplate error: %v", err)
	}
}

type FlashcardResult struct {
	MainWord    string
	LangResults []LangResult
}

type LangResult struct {
	Name    string
	Chosen  string
	Correct string
	Status  string // "correct" или "incorrect"
}

func FlashcardsCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := config.Store.Get(r, config.SessionName)
	if err != nil || session.Values["authenticated"] != true {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Получаем параметры из формы
	deckID := r.FormValue("deck_id")
	mainLangID := r.FormValue("main_lang_id")

	// Получаем название основного языка
	var mainLangTitle string
	err = db.Raw("SELECT lang_title FROM user_langs WHERE id = ?", mainLangID).Scan(&mainLangTitle).Error
	if err != nil {
		http.Error(w, "Failed to get main language title", http.StatusInternalServerError)
		return
	}

	// Получаем все языки колоды кроме основного
	var otherLangs []struct {
		ID    uint
		Title string
	}
	err = db.Raw(`
        SELECT ul.id, ul.lang_title as title 
        FROM deck_langs dl
        JOIN user_langs ul ON dl.lang_id = ul.id
        WHERE dl.deck_id = ? AND ul.id != ?
    `, deckID, mainLangID).Scan(&otherLangs).Error
	if err != nil {
		http.Error(w, "Failed to get other languages", http.StatusInternalServerError)
		return
	}

	// Собираем названия языков для заголовков таблицы
	var langTitles []string
	for _, lang := range otherLangs {
		langTitles = append(langTitles, lang.Title)
	}

	// Обрабатываем ответы
	var results []FlashcardResult

	// Проходим по всем словам в форме
	for key, values := range r.Form {
		if strings.HasPrefix(key, "word_") && strings.HasSuffix(key, "_main") {
			// Извлекаем ID слова
			wordIDStr := strings.TrimPrefix(strings.TrimSuffix(key, "_main"), "word_")
			mainWord := values[0]

			result := FlashcardResult{
				MainWord: mainWord,
			}

			// Проверяем ответы для каждого языка
			for _, lang := range otherLangs {
				answerKey := fmt.Sprintf("word_%s_lang_%d", wordIDStr, lang.ID)
				chosenAnswer := r.FormValue(answerKey)
				correctAnswerKey := fmt.Sprintf("word_%s_lang_%d_correct", wordIDStr, lang.ID)
				correctAnswer := r.FormValue(correctAnswerKey)

				status := "incorrect"
				if chosenAnswer == correctAnswer {
					status = "correct"
				}

				result.LangResults = append(result.LangResults, LangResult{
					Name:    lang.Title,
					Chosen:  chosenAnswer,
					Correct: correctAnswer,
					Status:  status,
				})
			}

			results = append(results, result)
		}
	}

	// Рендерим страницу с результатами
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/flashcardsCheck.html")
	if err != nil {
		log.Println("Template parse error:", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title         string
		MainLangTitle string
		LangTitles    []string
		Results       []FlashcardResult
	}{
		Title:         "Flashcards Results",
		MainLangTitle: mainLangTitle,
		LangTitles:    langTitles,
		Results:       results,
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Println("ExecuteTemplate error:", err)
		http.Error(w, "Render error", http.StatusInternalServerError)
		return
	}
}
