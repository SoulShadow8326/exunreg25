package handlers

import (
	"encoding/json"
	"exunreg25/db"
	"net/http"
)

type SummaryResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type EventSummary struct {
	EventID      string           `json:"event_id"`
	EventName    string           `json:"event_name"`
	Participants []db.Participant `json:"participants"`
	TotalCount   int              `json:"total_count"`
}

func GetUserSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response := SummaryResponse{
			Status:  "error",
			Message: "Method not allowed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !globalAuthHandler.isAuthenticated(r) {
		response := SummaryResponse{
			Status:  "error",
			Message: "Authentication required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	email := globalAuthHandler.getAuthenticatedUser(r)
	userData, err := globalDB.Get("users", email)
	if err != nil {
		response := SummaryResponse{
			Status:  "error",
			Message: "User not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	user := userData.(*db.User)
	if user.Username == "" {
		response := SummaryResponse{
			Status:  "error",
			Message: "Complete signup required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	if user.Registrations == nil {
		user.Registrations = make(map[string][]db.Participant)
	}

	totalRegistrations := len(user.Registrations)
	eventSummaries := []EventSummary{}
	totalParticipants := 0

	for eventID, participants := range user.Registrations {
		if len(participants) == 0 {
			continue
		}

		eventData, err := globalDB.Get("events", eventID)
		if err != nil {
			continue
		}

		event := eventData.(*db.Event)
		eventSummary := EventSummary{
			EventID:      eventID,
			EventName:    event.Name,
			Participants: participants,
			TotalCount:   len(participants),
		}

		eventSummaries = append(eventSummaries, eventSummary)
		totalParticipants += len(participants)
	}

	summaryData := map[string]interface{}{
		"total_events_registered": totalRegistrations,
		"total_participants":      totalParticipants,
		"events":                  eventSummaries,
		"user_info": map[string]interface{}{
			"fullname":         user.Fullname,
			"email":            user.Email,
			"institution_name": user.InstitutionName,
			"individual":       user.Individual,
		},
	}

	if totalRegistrations == 0 {
		summaryData["message"] = "No registrations found"
	} else {
		summaryData["message"] = "Registration summary retrieved successfully"
	}

	response := SummaryResponse{
		Status:  "success",
		Message: "Registration summary",
		Data:    summaryData,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}