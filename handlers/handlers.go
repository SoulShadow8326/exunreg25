package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"exunreg25/db"
)

func slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := Response{
		Status:  "ok",
		Message: "Server is running",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var userData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		response := Response{
			Status: "error",
			Error:  "Invalid request body",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	user := &db.User{
		Username:     userData.Username,
		Email:        userData.Email,
		PasswordHash: "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := globalDB.Create("users", user); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to create user",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Status:  "success",
		Message: "User registered successfully",
		Data:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		response := Response{
			Status: "error",
			Error:  "Invalid request body",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	userData, err := globalDB.Get("users", loginData.Email)
	if err != nil {
		response := Response{
			Status: "error",
			Error:  "User not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	user := userData.(*db.User)
	hashed := sha256.Sum256([]byte(os.Getenv("AUTH_SALT") + loginData.Password))
	if user.PasswordHash == "" || user.PasswordHash != hex.EncodeToString(hashed[:]) {
		response := Response{
			Status: "error",
			Error:  "Invalid credentials",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	authToken := sha256.Sum256([]byte(loginData.Email + os.Getenv("AUTH_SALT")))
	tokenStr := hex.EncodeToString(authToken[:])
	http.SetCookie(w, &http.Cookie{
		Name:     "email",
		Value:    loginData.Email,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	response := Response{
		Status:  "success",
		Message: "Login successful",
		Data: map[string]interface{}{
			"email": loginData.Email,
			"token": tokenStr,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		response := Response{
			Status: "error",
			Error:  "Email parameter required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	user, err := globalDB.Get("users", email)
	if err != nil {
		response := Response{
			Status: "error",
			Error:  "User not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Status:  "success",
		Message: "User profile retrieved successfully",
		Data:    user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetAllEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventsRaw, err := globalDB.GetAll("events")
	if err != nil {
		response := Response{Status: "error", Error: "Failed to retrieve events from DB"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	events := []map[string]interface{}{}
	for _, e := range eventsRaw {
		if ev, ok := e.(*db.Event); ok {
			event := map[string]interface{}{
				"id":                ev.ID,
				"name":              ev.Name,
				"image":             ev.Image,
				"slug":              ev.ID,
				"description_short": ev.DescriptionShort,
				"description_long":  ev.DescriptionLong,
				"participants":      ev.Participants,
				"mode":              ev.Mode,
				"points":            ev.Points,
				"individual":        ev.IndependentRegistration,
				"eligibility":       ev.Eligibility,
				"open_to_all":       ev.OpenToAll,
				"dates":             ev.Dates,
			}
			events = append(events, event)
		}
	}

	response := Response{Status: "success", Message: "Events retrieved successfully", Data: events}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventID := r.URL.Query().Get("id")
	if eventID == "" {
		response := Response{
			Status: "error",
			Error:  "Event ID parameter required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if ev, err := globalDB.Get("events", eventID); err == nil && ev != nil {
		if dbEv, ok := ev.(*db.Event); ok {
			foundEvent := map[string]interface{}{
				"id":                dbEv.ID,
				"name":              dbEv.Name,
				"image":             dbEv.Image,
				"slug":              dbEv.ID,
				"description_short": dbEv.DescriptionShort,
				"description_long":  dbEv.DescriptionLong,
				"participants":      dbEv.Participants,
				"mode":              dbEv.Mode,
				"points":            dbEv.Points,
				"individual":        dbEv.IndependentRegistration,
				"eligibility":       dbEv.Eligibility,
				"open_to_all":       dbEv.OpenToAll,
				"dates":             dbEv.Dates,
			}
			response := Response{Status: "success", Message: "Event retrieved successfully", Data: foundEvent}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	all, err := globalDB.GetAll("events")
	if err == nil {
		for _, item := range all {
			if dbEv, ok := item.(*db.Event); ok {
				if dbEv.ID == eventID || slugify(dbEv.Name) == eventID {
					foundEvent := map[string]interface{}{
						"id":                dbEv.ID,
						"name":              dbEv.Name,
						"image":             dbEv.Image,
						"slug":              dbEv.ID,
						"description_short": dbEv.DescriptionShort,
						"description_long":  dbEv.DescriptionLong,
						"participants":      dbEv.Participants,
						"mode":              dbEv.Mode,
						"points":            dbEv.Points,
						"individual":        dbEv.IndependentRegistration,
						"eligibility":       dbEv.Eligibility,
						"open_to_all":       dbEv.OpenToAll,
						"dates":             dbEv.Dates,
					}
					response := Response{Status: "success", Message: "Event retrieved successfully", Data: foundEvent}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
					return
				}
			}
		}
	}

	response := Response{Status: "error", Error: "Event not found"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(response)
}

var globalDB *db.Database

func SetGlobalDB(database *db.Database) {
	globalDB = database
}

func GetAllEventsData() ([]db.Event, error) {
	events := []db.Event{}
	all, err := globalDB.GetAll("events")
	if err != nil {
		return nil, err
	}
	for _, item := range all {
		if ev, ok := item.(*db.Event); ok {
			events = append(events, *ev)
		}
	}
	return events, nil
}
