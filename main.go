package main

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Participant struct {
	Name      string `json:"name"`
	Wish      string `json:"wish"`
	GiftFor   string `json:"giftFor"`
	Submitted bool   `json:"submitted"`
}

type Draw struct {
	Name                 string                  `json:"name"`
	ExpectedParticipants *int                    `json:"expectedParticipants"`
	Participants         map[string]*Participant `json:"participants"`
	DrawDone             bool                    `json:"drawDone"`
	CreatedAt            time.Time               `json:"createdAt"`
}

type Data struct {
	Events map[string]*Draw `json:"events"`
}

type Translations map[string]string

var templates = template.Must(template.ParseGlob("templates/*.html"))
var dataFile = "data.json"
var appData Data
var dataMutex sync.RWMutex

const (
	maxNameLength   = 100
	maxWishLength   = 500
	maxActiveEvents = 1000
)

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() string {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := cryptorand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

// validateInput sanitizes and validates user input
func validateInput(input string, maxLength int, fieldName string) (string, error) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check if empty
	if input == "" {
		return "", fmt.Errorf("%s cannot be empty", fieldName)
	}

	// Check length
	if len(input) > maxLength {
		return "", fmt.Errorf("%s is too long (max %d characters)", fieldName, maxLength)
	}

	return input, nil
}

func main() {
	mathrand.Seed(time.Now().UnixNano())
	loadData()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/draw/create", createDrawHandler)
	http.HandleFunc("/draw/", drawHandler)

	// Get port from environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server started at http://localhost:%s\n", port)

	mux := http.DefaultServeMux

	// forceHTTPS redirects HTTP -> HTTPS for non-local requests using a 301.
	// We intentionally allow localhost/127.0.0.1 to remain on HTTP for local dev.
	forceHTTPS := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isHTTPS(r) && !strings.HasPrefix(r.Host, "localhost") && !strings.HasPrefix(r.Host, "127.0.0.1") {
				url := "https://" + r.Host + r.URL.RequestURI()
				http.Redirect(w, r, url, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	handler := forceHTTPS(mux)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if strings.EqualFold(proto, "https") {
		return true
	}
	return false
}

func loadData() {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	file, err := os.Open(dataFile)
	if err != nil {
		fmt.Println("Data file not found, creating new one.")
		appData.Events = make(map[string]*Draw)
		return
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading data file: %v", err)
		appData.Events = make(map[string]*Draw)
		return
	}

	if err := json.Unmarshal(bytes, &appData); err != nil {
		log.Printf("Error parsing data file: %v", err)
		appData.Events = make(map[string]*Draw)
		return
	}

	cleanupOldEvents()
}

// cleanupOldEvents removes draws older than 30 days
// Note: This function should be called when dataMutex is already locked
func cleanupOldEvents() {
	cutoffDate := time.Now().AddDate(0, 0, -30)
	deleted := 0
	for id, draw := range appData.Events {
		if draw.CreatedAt.Before(cutoffDate) {
			delete(appData.Events, id)
			deleted++
		}
	}
	if deleted > 0 {
		fmt.Printf("Cleaned up %d old draws (older than 30 days)\n", deleted)
		saveDataUnsafe()
	}
}

func saveData() {
	dataMutex.Lock()
	defer dataMutex.Unlock()
	saveDataUnsafe()
}

// saveDataUnsafe saves data without acquiring the mutex (for when already locked)
func saveDataUnsafe() {
	bytes, err := json.MarshalIndent(appData, "", "  ")
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return
	}

	if err := os.WriteFile(dataFile, bytes, 0644); err != nil {
		log.Printf("Error writing data file: %v", err)
	}
}

func getLanguage(r *http.Request) string {
	// Check query parameter first (for manual override)
	lang := r.URL.Query().Get("lang")
	if lang != "" {
		return lang
	}

	// Parse Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// Accept-Language format: "en-US,en;q=0.9,fr;q=0.8"
		langs := parseAcceptLanguage(acceptLang)
		for _, l := range langs {
			// Check if we support this language
			if l == "en" || l == "fr" || l == "de" || l == "pt" {
				return l
			}
		}
	}

	// Default to English
	return "en"
}

func parseAcceptLanguage(header string) []string {
	var langs []string
	for _, part := range splitByComma(header) {
		// Split by semicolon to remove quality values (;q=0.9)
		langPart := part
		if idx := indexByte(part, ';'); idx != -1 {
			langPart = part[:idx]
		}
		// Trim spaces and extract base language (en-US -> en)
		langPart = trimSpace(langPart)
		if idx := indexByte(langPart, '-'); idx != -1 {
			langPart = langPart[:idx]
		}
		if langPart != "" {
			langs = append(langs, langPart)
		}
	}
	return langs
}

func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func loadTranslations(lang string) Translations {
	if lang == "" {
		lang = "en"
	}
	filename := fmt.Sprintf("locales/%s.json", lang)
	file, err := os.Open(filename)
	if err != nil {
		file, _ = os.Open("locales/en.json") // fallback
	}
	if file != nil {
		defer file.Close()
	}
	bytes, _ := io.ReadAll(file)
	var t Translations
	json.Unmarshal(bytes, &t)
	return t
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/draw/create", http.StatusSeeOther)
}

func createDrawHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLanguage(r)
	t := loadTranslations(lang)

	if r.Method == http.MethodGet {
		canonical := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
		templates.ExecuteTemplate(w, "create_event.html", struct {
			T           Translations
			CurrentLang string
			Canonical   string
		}{t, lang, canonical})
		return
	}
	r.ParseForm()
	eventName := r.FormValue("eventname")
	organizerName := r.FormValue("organizername")
	organizerWish := r.FormValue("organizerwish")
	expected := r.FormValue("expected")

	// Validate inputs
	eventName, err := validateInput(eventName, maxNameLength, "Draw name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	organizerName, err = validateInput(organizerName, maxNameLength, "Organizer name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Wish is optional but has max length if provided
	if organizerWish != "" {
		if len(organizerWish) > maxWishLength {
			http.Error(w, fmt.Sprintf("Wish is too long (max %d characters)", maxWishLength), http.StatusBadRequest)
			return
		}
	}

	// Validate expected participants
	expectedNum := 0
	fmt.Sscanf(expected, "%d", &expectedNum)
	if expectedNum < 3 || expectedNum > 50 {
		http.Error(w, "Expected participants must be between 3 and 50", http.StatusBadRequest)
		return
	}

	// Check if we've hit the max active events limit
	dataMutex.RLock()
	activeEvents := len(appData.Events)
	dataMutex.RUnlock()

	if activeEvents >= maxActiveEvents {
		http.Error(w, "Server is at capacity. Please try again later.", http.StatusServiceUnavailable)
		return
	}

	id := generateSecureToken()
	organizerToken := generateSecureToken()

	dataMutex.Lock()
	appData.Events[id] = &Draw{
		Name:                 eventName,
		ExpectedParticipants: &expectedNum,
		Participants: map[string]*Participant{
			organizerToken: {
				Name:      organizerName,
				Wish:      organizerWish,
				Submitted: true,
			},
		},
		DrawDone:  false,
		CreatedAt: time.Now(),
	}
	dataMutex.Unlock()
	saveData()

	// Redirect to manage page with organizer's participant token in query
	http.Redirect(w, r, "/draw/"+id+"/manage?organizer="+organizerToken, http.StatusSeeOther)
}

func drawHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/draw/"):] // /{id}/...

	// Extract draw ID (everything before the first slash or the whole string)
	var id string
	slashIndex := strings.Index(path, "/")
	if slashIndex == -1 {
		id = path
	} else {
		id = path[:slashIndex]
	}

	dataMutex.RLock()
	draw, ok := appData.Events[id]
	dataMutex.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	lang := getLanguage(r)
	t := loadTranslations(lang)

	// Extract action from path (e.g., "join", "manage", "participant/{token}", "draw")
	action := ""
	if slashIndex != -1 && slashIndex+1 < len(path) {
		action = path[slashIndex+1:]
	}

	// Handle participant/{token} specially
	if len(action) > 12 && action[:12] == "participant/" {
		token := action[12:] // Extract token after "participant/"

		dataMutex.RLock()
		p, ok := draw.Participants[token]
		dataMutex.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		if !draw.DrawDone {
			canonical := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
			templates.ExecuteTemplate(w, "participant.html", struct {
				Name        string
				Ready       bool
				T           Translations
				CurrentLang string
				Canonical   string
			}{p.Name, false, t, lang, canonical})
		} else {
			// Find the wish of the person they're giving a gift to
			recipientWish := ""
			for _, participant := range draw.Participants {
				if participant.Name == p.GiftFor {
					recipientWish = participant.Wish
					break
				}
			}
			canonical := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
			templates.ExecuteTemplate(w, "participant.html", struct {
				Name        string
				Ready       bool
				GiftFor     string
				Wish        string
				T           Translations
				CurrentLang string
				Canonical   string
			}{p.Name, true, p.GiftFor, recipientWish, t, lang, canonical})
		}
		return
	}

	switch action {
	case "join":
		if r.Method == http.MethodGet {
			canonical := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
			templates.ExecuteTemplate(w, "join.html", struct {
				EventID     string
				T           Translations
				CurrentLang string
				Canonical   string
			}{id, t, lang, canonical})
			return
		}
		r.ParseForm()

		// Check if draw has reached participant limit
		dataMutex.RLock()
		isFull := draw.ExpectedParticipants != nil && len(draw.Participants) >= *draw.ExpectedParticipants
		dataMutex.RUnlock()

		if isFull {
			http.Error(w, "Draw is full - maximum participants reached", http.StatusForbidden)
			return
		}

		name := r.FormValue("name")
		wish := r.FormValue("wish")

		// Validate inputs
		name, err := validateInput(name, maxNameLength, "Name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Wish is optional but has max length if provided
		if wish != "" {
			if len(wish) > maxWishLength {
				http.Error(w, fmt.Sprintf("Wish is too long (max %d characters)", maxWishLength), http.StatusBadRequest)
				return
			}
		}

		token := generateSecureToken()

		dataMutex.Lock()
		draw.Participants[token] = &Participant{Name: name, Wish: wish, Submitted: true}
		dataMutex.Unlock()

		saveData()
		http.Redirect(w, r, "/draw/"+id+"/participant/"+token, http.StatusSeeOther)

	case "manage":
		dataMutex.RLock()
		allSubmitted := true
		for _, part := range draw.Participants {
			if !part.Submitted {
				allSubmitted = false
				break
			}
		}

		// Check if expected number of participants is reached
		expectedReached := false
		if draw.ExpectedParticipants != nil {
			expectedReached = len(draw.Participants) >= *draw.ExpectedParticipants
		}
		dataMutex.RUnlock()

		// Build canonical links using HTTPS
		scheme := "https"
		joinLink := fmt.Sprintf(scheme+"://%s/draw/%s/join", r.Host, id)
		organizerToken := r.URL.Query().Get("organizer")
		organizerLink := ""
		// Only show organizer link after draw is done
		if organizerToken != "" && draw.DrawDone {
			organizerLink = fmt.Sprintf(scheme+"://%s/draw/%s/participant/%s", r.Host, id, organizerToken)
		}
		canDraw := allSubmitted && !draw.DrawDone && expectedReached
		canonical := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
		templates.ExecuteTemplate(w, "manage.html", struct {
			EventID        string
			EventName      string
			JoinLink       string
			OrganizerLink  string
			OrganizerToken string
			Participants   map[string]*Participant
			CanDraw        bool
			DrawDone       bool
			T              Translations
			CurrentLang    string
			Canonical      string
		}{id, draw.Name, joinLink, organizerLink, organizerToken, draw.Participants, canDraw, draw.DrawDone, t, lang, canonical})

	case "draw":
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		dataMutex.Lock()
		defer dataMutex.Unlock()

		// Need at least 3 participants for a proper Secret Santa
		if len(draw.Participants) < 3 {
			http.Error(w, "Need at least 3 participants", http.StatusBadRequest)
			return
		}

		tokens := make([]string, 0, len(draw.Participants))
		for t := range draw.Participants {
			tokens = append(tokens, t)
		}
		mathrand.Shuffle(len(tokens), func(i, j int) { tokens[i], tokens[j] = tokens[j], tokens[i] })
		n := len(tokens)
		for i, t := range tokens {
			next := tokens[(i+1)%n]
			draw.Participants[t].GiftFor = draw.Participants[next].Name
		}
		draw.DrawDone = true
		saveDataUnsafe()

		// Redirect back to manage page, preserving organizer token if present
		organizerToken := r.URL.Query().Get("organizer")
		redirectURL := "/draw/" + id + "/manage"
		if organizerToken != "" {
			redirectURL += "?organizer=" + organizerToken
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)

	default:
		http.NotFound(w, r)
	}
}
