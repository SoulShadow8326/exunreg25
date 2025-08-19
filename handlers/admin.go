package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"exunreg25/db"
)

type AdminHandler struct {
	db *db.Database
}

type EventUpdateRequest struct {
	EventID                 string `json:"event_id"`
	Mode                    string `json:"mode"`
	Participants            int    `json:"participants"`
	MinClass                int    `json:"min_class"`
	MaxClass                int    `json:"max_class"`
	OpenToAll               bool   `json:"open_to_all"`
	IndependentRegistration bool   `json:"independent_registration"`
	Points                  int    `json:"points"`
	Dates                   string `json:"dates"`
	DescriptionShort        string `json:"description_short"`
	DescriptionLong         string `json:"description_long"`
}

type AdminStats struct {
	TotalUsers         int                   `json:"total_users"`
	TotalEvents        int                   `json:"total_events"`
	TotalRegistrations int                   `json:"total_registrations"`
	EventStats         map[string]EventStats `json:"event_stats"`
	UserRegistrations  map[string]UserStats  `json:"user_registrations"`
}

type EventStats struct {
	EventName         string `json:"event_name"`
	TotalParticipants int    `json:"total_participants"`
	TotalTeams        int    `json:"total_teams"`
	Mode              string `json:"mode"`
	Eligibility       string `json:"eligibility"`
}

type UserStats struct {
	Email             string `json:"email"`
	Fullname          string `json:"fullname"`
	Institution       string `json:"institution"`
	TotalEvents       int    `json:"total_events"`
	TotalParticipants int    `json:"total_participants"`
}

func NewAdminHandler(database *db.Database) *AdminHandler {
	return &AdminHandler{
		db: database,
	}
}

func (ah *AdminHandler) AdminPanel(w http.ResponseWriter, r *http.Request) {
	if globalAuthHandler == nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "./frontend/admin.html")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Admin panel access granted",
		"status":  "success",
	})
}

func (ah *AdminHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	users, err := ah.db.GetAll("users")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	events, err := ah.db.GetAll("events")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	stats := AdminStats{
		TotalUsers:         len(users),
		TotalEvents:        len(events),
		TotalRegistrations: 0,
		EventStats:         make(map[string]EventStats),
		UserRegistrations:  make(map[string]UserStats),
	}

	for _, eventData := range events {
		event := eventData.(db.Event)
		eventStats := EventStats{
			EventName:         event.Name,
			TotalParticipants: 0,
			TotalTeams:        0,
			Mode:              event.Mode,
			Eligibility:       event.Eligibility,
		}

		for _, userData := range users {
			user := userData.(db.User)
			if participants, exists := user.Registrations[event.ID]; exists {
				eventStats.TotalParticipants += len(participants)
				eventStats.TotalTeams++
				stats.TotalRegistrations += len(participants)
			}
		}

		stats.EventStats[event.ID] = eventStats
	}

	for _, userData := range users {
		user := userData.(db.User)
		userStats := UserStats{
			Email:             user.Email,
			Fullname:          user.Fullname,
			Institution:       user.InstitutionName,
			TotalEvents:       len(user.Registrations),
			TotalParticipants: 0,
		}

		for _, participants := range user.Registrations {
			userStats.TotalParticipants += len(participants)
		}

		stats.UserRegistrations[user.Email] = userStats
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (ah *AdminHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := strings.TrimPrefix(r.URL.Path, "/api/admin/events/")
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	eventData, err := ah.db.Get("events", eventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	event := eventData.(db.Event)
	response := map[string]interface{}{
		"id":                       event.ID,
		"mode":                     event.Mode,
		"participants":             event.Participants,
		"eligibility":              event.Eligibility,
		"open_to_all":              event.OpenToAll,
		"independent_registration": event.IndependentRegistration,
		"points":                   event.Points,
		"dates":                    event.Dates,
		"descriptions": map[string]string{
			"short": event.DescriptionShort,
			"long":  event.DescriptionLong,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ah *AdminHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req EventUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existingEventData, err := ah.db.Get("events", req.EventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	existingEvent := existingEventData.(db.Event)
	eligibilityStr := fmt.Sprintf("[%d,%d]", req.MinClass, req.MaxClass)

	updatedEvent := db.Event{
		ID:                      req.EventID,
		Name:                    existingEvent.Name,
		Image:                   existingEvent.Image,
		Mode:                    req.Mode,
		Participants:            req.Participants,
		Eligibility:             eligibilityStr,
		OpenToAll:               req.OpenToAll,
		IndependentRegistration: req.IndependentRegistration,
		Points:                  req.Points,
		Dates:                   req.Dates,
		DescriptionShort:        req.DescriptionShort,
		DescriptionLong:         req.DescriptionLong,
		CreatedAt:               existingEvent.CreatedAt,
		UpdatedAt:               time.Now(),
	}

	if err := ah.db.Update("events", req.EventID, updatedEvent); err != nil {
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Event updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ah *AdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := strings.TrimPrefix(r.URL.Path, "/api/admin/events/")
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	if err := ah.db.Delete("events", eventID); err != nil {
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Event deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ah *AdminHandler) GetUserDetails(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userEmail := r.URL.Query().Get("email")
	if userEmail == "" {
		http.Error(w, "Email parameter required", http.StatusBadRequest)
		return
	}

	userData, err := ah.db.Get("users", userEmail)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userData)
}

func (ah *AdminHandler) GetEventRegistrations(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		http.Error(w, "Event ID parameter required", http.StatusBadRequest)
		return
	}

	users, err := ah.db.GetAll("users")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var registrations []map[string]interface{}
	for _, userData := range users {
		user := userData.(db.User)
		if participants, exists := user.Registrations[eventID]; exists {
			for _, participant := range participants {
				registrations = append(registrations, map[string]interface{}{
					"user_email":  user.Email,
					"user_name":   user.Fullname,
					"institution": user.InstitutionName,
					"participant": participant,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registrations)
}

func (ah *AdminHandler) ExportData(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	exportType := r.URL.Query().Get("type")
	if exportType == "" {
		exportType = "all"
	}

	var data interface{}
	var filename string

	switch exportType {
	case "users":
		users, _ := ah.db.GetAll("users")
		data = users
		filename = "users_export.json"
	case "events":
		events, _ := ah.db.GetAll("events")
		data = events
		filename = "events_export.json"
	case "registrations":
		users, _ := ah.db.GetAll("users")
		registrations := make(map[string]interface{})
		for _, userData := range users {
			user := userData.(db.User)
			registrations[user.Email] = user.Registrations
		}
		data = registrations
		filename = "registrations_export.json"
	default:
		users, _ := ah.db.GetAll("users")
		events, _ := ah.db.GetAll("events")
		data = map[string]interface{}{
			"users":       users,
			"events":      events,
			"exported_at": time.Now().Format(time.RFC3339),
		}
		filename = "full_export.json"
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(jsonData)
}

var globalAdminHandler *AdminHandler

func SetGlobalAdminHandler(handler *AdminHandler) {
	globalAdminHandler = handler
}

func AdminPanel(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.AdminPanel(w, r)
}

func GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.GetAdminStats(w, r)
}

func GetAdminEvent(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.GetEvent(w, r)
}

func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.UpdateEvent(w, r)
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.DeleteEvent(w, r)
}

func GetUserDetails(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.GetUserDetails(w, r)
}

func GetEventRegistrations(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.GetEventRegistrations(w, r)
}

func ExportData(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.ExportData(w, r)
}

func IsAdminEmail(email string) bool {
	if email == "" {
		return false
	}
	admins := os.Getenv("ADMIN_EMAILS")
	if admins == "" {
		admins = os.Getenv("ADMIN_EMAIL")
	}
	if admins == "" {
		admins = "exun@dpsrkp.net"
	}
	for _, a := range strings.Split(admins, ",") {
		if strings.TrimSpace(strings.ToLower(a)) == strings.ToLower(email) {
			return true
		}
	}
	return false
}

func (ah *AdminHandler) GetAdminConfig(w http.ResponseWriter, r *http.Request) {
	admins := os.Getenv("ADMIN_EMAILS")
	if admins == "" {
		admins = os.Getenv("ADMIN_EMAIL")
	}
	if admins == "" {
		admins = "exun@dpsrkp.net"
	}
	resp := map[string]interface{}{
		"admin_emails": admins,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetAdminConfig(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.GetAdminConfig(w, r)
}
