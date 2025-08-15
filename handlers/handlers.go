package handlers

import (
	"bytes"
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

	file, err := http.Dir("./frontend/data").Open("events.json")
	if err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to read events.json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer file.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to read events.json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &top); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to parse events.json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	var descriptions map[string]map[string]interface{}
	var participants map[string]interface{}
	var modes map[string]interface{}
	var points map[string]interface{}
	var individual map[string]interface{}
	var eligibility map[string]interface{}
	var openToAll map[string]interface{}

	json.Unmarshal(top["descriptions"], &descriptions)
	json.Unmarshal(top["participants"], &participants)
	json.Unmarshal(top["mode"], &modes)
	json.Unmarshal(top["points"], &points)
	json.Unmarshal(top["individual"], &individual)
	json.Unmarshal(top["eligibility"], &eligibility)
	json.Unmarshal(top["open_to_all"], &openToAll)

	events := []map[string]interface{}{}
	if eventsRaw, ok := top["events"]; ok {
		dec := json.NewDecoder(bytes.NewReader(eventsRaw))
		if tok, err := dec.Token(); err == nil {
			if delim, ok := tok.(json.Delim); ok && delim == '{' {
				for dec.More() {
					kTok, err := dec.Token()
					if err != nil {
						break
					}
					name, _ := kTok.(string)
					var image string
					if err := dec.Decode(&image); err != nil {
						image = ""
					}

					event := map[string]interface{}{
						"id":    name,
						"name":  name,
						"image": image,
						"slug":  slugify(name),
					}

					if descMap, exists := descriptions[name]; exists {
						event["description_short"] = descMap["short"]
						event["description_long"] = descMap["long"]
					}

					if p, exists := participants[name]; exists {
						event["participants"] = p
					}

					if m, exists := modes[name]; exists {
						event["mode"] = m
					}

					if pt, exists := points[name]; exists {
						event["points"] = pt
					}

					if ind, exists := individual[name]; exists {
						event["individual"] = ind
					}

					if elig, exists := eligibility[name]; exists {
						event["eligibility"] = elig
					}

					if open, exists := openToAll[name]; exists {
						event["open_to_all"] = open
					}

					events = append(events, event)
				}
				dec.Token()
			}
		}
	}

	response := Response{
		Status:  "success",
		Message: "Events retrieved successfully",
		Data:    events,
	}

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

	file, err := http.Dir("./frontend/data").Open("events.json")
	if err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to read events.json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to read events.json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &top); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to parse events.json",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	var descriptions map[string]map[string]interface{}
	var participants map[string]interface{}
	var modes map[string]interface{}
	var points map[string]interface{}
	var individual map[string]interface{}
	var eligibility map[string]interface{}
	var openToAll map[string]interface{}

	json.Unmarshal(top["descriptions"], &descriptions)
	json.Unmarshal(top["participants"], &participants)
	json.Unmarshal(top["mode"], &modes)
	json.Unmarshal(top["points"], &points)
	json.Unmarshal(top["individual"], &individual)
	json.Unmarshal(top["eligibility"], &eligibility)
	json.Unmarshal(top["open_to_all"], &openToAll)

	var eventMap map[string]string
	json.Unmarshal(top["events"], &eventMap)

	var foundEvent map[string]interface{}
	for name, image := range eventMap {
		if name == eventID || slugify(name) == eventID {
			foundEvent = map[string]interface{}{
				"id":    name,
				"name":  name,
				"image": image,
			}

			if descMap, exists := descriptions[name]; exists {
				if descMap != nil {
					foundEvent["description_short"] = descMap["short"]
					foundEvent["description_long"] = descMap["long"]
				}
			}

			if p, exists := participants[name]; exists {
				foundEvent["participants"] = p
			}

			if m, exists := modes[name]; exists {
				foundEvent["mode"] = m
			}

			if pt, exists := points[name]; exists {
				foundEvent["points"] = pt
			}

			if ind, exists := individual[name]; exists {
				foundEvent["individual"] = ind
			}

			if elig, exists := eligibility[name]; exists {
				foundEvent["eligibility"] = elig
			}

			if open, exists := openToAll[name]; exists {
				foundEvent["open_to_all"] = open
			}

			break
		}
	}

	if foundEvent == nil {
		response := Response{
			Status: "error",
			Error:  "Event not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Status:  "success",
		Message: "Event retrieved successfully",
		Data:    foundEvent,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

var globalDB *db.Database

func SetGlobalDB(database *db.Database) {
	globalDB = database
}
