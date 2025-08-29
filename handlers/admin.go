package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"exunreg25/db"
	"exunreg25/mail"
)

type AdminHandler struct {
	db *db.Database
}

type InvitePayload struct {
	ToEmail       string `json:"to_email"`
	SchoolName    string `json:"school_name"`
	PrincipalName string `json:"principal_name,omitempty"`
	CustomMessage string `json:"custom_message,omitempty"`
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
	stats, err := GetAdminStatsData()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func GetAdminStatsData() (AdminStats, error) {
	users, err := globalDB.GetAll("users")
	if err != nil {
		return AdminStats{}, err
	}

	events, err := globalDB.GetAll("events")
	if err != nil {
		return AdminStats{}, err
	}

	var nonAdminUsers []interface{}
	for _, u := range users {
		user := u.(*db.User)
		if IsAdminEmail(user.Email) {
			continue
		}
		nonAdminUsers = append(nonAdminUsers, u)
	}

	stats := AdminStats{
		TotalUsers:         len(nonAdminUsers),
		TotalEvents:        len(events),
		TotalRegistrations: 0,
		EventStats:         make(map[string]EventStats),
		UserRegistrations:  make(map[string]UserStats),
	}

	for _, eventData := range events {
		event := eventData.(*db.Event)
		eventStats := EventStats{
			EventName:         event.Name,
			TotalParticipants: 0,
			TotalTeams:        0,
			Mode:              event.Mode,
			Eligibility:       event.Eligibility,
		}

		for _, userData := range nonAdminUsers {
			user := userData.(*db.User)
			if participants, exists := user.Registrations[event.ID]; exists {
				eventStats.TotalParticipants += len(participants)
				eventStats.TotalTeams++
				stats.TotalRegistrations++
			}
		}

		stats.EventStats[event.ID] = eventStats
	}

	for _, userData := range nonAdminUsers {
		user := userData.(*db.User)
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

	return stats, nil
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

	event := eventData.(*db.Event)
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

	existingEvent := existingEventData.(*db.Event)
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

func (ah *AdminHandler) SendInvite(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InvitePayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ToEmail != "" {
		u := db.User{
			Email:           req.ToEmail,
			InstitutionName: req.SchoolName,
			Fullname:        "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := ah.db.Create("users", &u); err != nil {
			if existing, err2 := ah.db.Get("users", req.ToEmail); err2 == nil {
				eu := existing.(db.User)
				if eu.InstitutionName == "" {
					eu.InstitutionName = req.SchoolName
					eu.UpdatedAt = time.Now()
					_ = ah.db.Update("users", req.ToEmail, eu)
				}
			}
		}
	}

	if inviteService != nil {
		mreq := mail.InviteEmailRequest{
			ToEmail:       req.ToEmail,
			SchoolName:    req.SchoolName,
			PrincipalName: req.PrincipalName,
			CustomMessage: req.CustomMessage,
		}
		if err := inviteService.SendInviteEmail(mreq); err != nil {
			http.Error(w, "Failed to send invite", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
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

	usersRaw, err := ah.db.GetAll("users")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	usersByID := make(map[int]db.User)
	for _, u := range usersRaw {
		switch usr := u.(type) {
		case *db.User:
			usersByID[usr.ID] = *usr
		case db.User:
			usersByID[usr.ID] = usr
		}
	}

	regsRaw, err := ah.db.GetAll("registrations")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	eventsRaw, _ := ah.db.GetAll("events")
	eventsByID := make(map[string]db.Event)
	for _, ev := range eventsRaw {
		switch e := ev.(type) {
		case *db.Event:
			eventsByID[e.ID] = *e
		case db.Event:
			eventsByID[e.ID] = e
		}
	}

	var out []map[string]interface{}
	for _, rr := range regsRaw {
		reg, ok := rr.(*db.Registration)
		if !ok {
			continue
		}
		if eventID != "" && reg.EventID != eventID {
			continue
		}

		user, found := usersByID[reg.UserID]
		if !found {
			if uiface, err := ah.db.Get("users", ""); err == nil && uiface != nil {
				_ = uiface
			}
			for _, ur := range usersRaw {
				switch uu := ur.(type) {
				case *db.User:
					if uu.ID == reg.UserID {
						user = *uu
						found = true
						break
					}
				case db.User:
					if uu.ID == reg.UserID {
						user = uu
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		userEmail := ""
		userName := ""
		if found {
			userEmail = user.Email
			userName = user.Fullname
		}
		teamName := reg.TeamName
		status := reg.Status
		created := reg.CreatedAt

		members := []db.Participant{}
		memberCount := 0
		if user.Registrations != nil {
			if parts, ok := user.Registrations[reg.EventID]; ok {
				members = parts
				memberCount = len(parts)
			}
		}

		eventName := ""
		if ev, ok := eventsByID[reg.EventID]; ok {
			eventName = ev.Name
		}

		out = append(out, map[string]interface{}{
			"eventId":     reg.EventID,
			"eventName":   eventName,
			"userEmail":   userEmail,
			"userName":    userName,
			"teamName":    teamName,
			"members":     members,
			"memberCount": memberCount,
			"createdAt":   created,
			"status":      status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
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
var inviteService *mail.InviteEmailService

func (ah *AdminHandler) ImportEvents(w http.ResponseWriter, r *http.Request) {
	if !globalAuthHandler.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	email := globalAuthHandler.getAuthenticatedUser(r)
	if !IsAdminEmail(email) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	b, err := os.ReadFile("frontend/data/events.json")
	if err != nil {
		http.Error(w, "Failed to read events.json", http.StatusInternalServerError)
		return
	}

	var raw struct {
		Default struct {
			OpenToAll    bool   `json:"open_to_all"`
			Eligibility  []int  `json:"eligibility"`
			Participants int    `json:"participants"`
			Mode         string `json:"mode"`
			Descriptions struct {
				Long  string `json:"long"`
				Short string `json:"short"`
			} `json:"descriptions"`
			IndependentRegistrations bool   `json:"independent_registrations"`
			Points                   int    `json:"points"`
			Dates                    string `json:"dates"`
		} `json:"default"`
		Descriptions map[string]struct {
			Long  string `json:"long"`
			Short string `json:"short"`
		} `json:"descriptions"`
		Participants map[string]int    `json:"participants"`
		Mode         map[string]string `json:"mode"`
		Points       map[string]int    `json:"points"`
		Individual   map[string]bool   `json:"individual"`
		Eligibility  map[string][]int  `json:"eligibility"`
		OpenToAll    map[string]bool   `json:"open_to_all"`
	}

	if err := json.Unmarshal(b, &raw); err != nil {
		http.Error(w, "Failed to parse events.json", http.StatusInternalServerError)
		return
	}

	type evtPair struct{ Name, Image string }
	ordered := []evtPair{}
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err == nil {
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				break
			}
			key, ok := k.(string)
			if !ok {
				var skip interface{}
				_ = dec.Decode(&skip)
				continue
			}
			if key == "events" {
				if _, err := dec.Token(); err != nil {
					break
				}
				for dec.More() {
					kn, err := dec.Token()
					if err != nil {
						break
					}
					name, _ := kn.(string)
					var img string
					if err := dec.Decode(&img); err != nil {
						break
					}
					ordered = append(ordered, evtPair{Name: name, Image: img})
				}
				_, _ = dec.Token()
			} else {
				var skip interface{}
				_ = dec.Decode(&skip)
			}
		}
		_ = tok
	}

	created := 0
	updated := 0
	for _, pair := range ordered {
		name := pair.Name
		image := pair.Image
		slug := slugify(name)
		participants := raw.Default.Participants
		if v, ok := raw.Participants[name]; ok {
			participants = v
		}
		mode := raw.Default.Mode
		if v, ok := raw.Mode[name]; ok {
			mode = v
		}
		points := raw.Default.Points
		if v, ok := raw.Points[name]; ok {
			points = v
		}
		individual := raw.Default.IndependentRegistrations
		if v, ok := raw.Individual[name]; ok {
			individual = v
		}

		descShort := raw.Default.Descriptions.Short
		descLong := raw.Default.Descriptions.Long
		if v, ok := raw.Descriptions[name]; ok {
			if v.Short != "" {
				descShort = v.Short
			}
			if v.Long != "" {
				descLong = v.Long
			}
		}

		openAll := raw.Default.OpenToAll
		if v, ok := raw.OpenToAll[name]; ok {
			openAll = v
		}

		eligibility := ""
		if openAll {
			eligibility = "Open to all"
		} else if vals, ok := raw.Eligibility[name]; ok && len(vals) >= 2 {
			eligibility = fmt.Sprintf("Grades %d–%d", vals[0], vals[1])
		} else if len(raw.Default.Eligibility) >= 2 {
			eligibility = fmt.Sprintf("Grades %d–%d", raw.Default.Eligibility[0], raw.Default.Eligibility[1])
		}

		ev := db.Event{
			ID:                      slug,
			Name:                    name,
			Image:                   image,
			OpenToAll:               openAll,
			Eligibility:             eligibility,
			Participants:            participants,
			Mode:                    mode,
			IndependentRegistration: individual,
			Points:                  points,
			Dates:                   raw.Default.Dates,
			DescriptionShort:        descShort,
			DescriptionLong:         descLong,
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		}

		if existing, err := ah.db.Get("events", slug); err == nil && existing != nil {
			_ = ah.db.Update("events", slug, ev)
			updated++
		} else {
			_ = ah.db.Create("events", &ev)
			created++
		}
	}

	resp := map[string]interface{}{"created": created, "updated": updated}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func SetInviteService(svc *mail.InviteEmailService) {
	inviteService = svc
}

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

func SendInvite(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.SendInvite(w, r)
}

func ImportEvents(w http.ResponseWriter, r *http.Request) {
	if globalAdminHandler == nil {
		http.Error(w, "Admin handler not initialized", http.StatusInternalServerError)
		return
	}
	globalAdminHandler.ImportEvents(w, r)
}
