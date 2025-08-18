package handlers

import (
	"encoding/json"
	"exunreg25/db"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Participant struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Class int    `json:"class"`
	Phone string `json:"phone"`
}

type RegistrationRequest struct {
	EventID string        `json:"id"`
	Data    []Participant `json:"data"`
}

type RegistrationResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func SubmitRegistrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response := Response{
			Status: "error",
			Error:  "Method not allowed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !globalAuthHandler.isAuthenticated(r) {
		response := Response{
			Status: "error",
			Error:  "Authentication required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	email := globalAuthHandler.getAuthenticatedUser(r)
	userData, err := globalDB.Get("users", email)
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
	if user.Username == "" {
		response := Response{
			Status: "error",
			Error:  "Complete signup required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	var req RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{
			Status: "error",
			Error:  "Invalid request format",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	eventData, err := globalDB.Get("events", req.EventID)
	var event *db.Event
	if err != nil {
		evt, jerr := loadEventFromJSON(req.EventID)
		if jerr != nil || evt == nil {
			response := Response{
				Status: "error",
				Error:  "Event not found",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}
		event = evt
	}

	if eventData != nil {
		if ev, ok := eventData.(*db.Event); ok {
			event = ev
		}
	}
	if !event.IndependentRegistration && user.Username != "" {
		response := Response{
			Status: "error",
			Error:  "Individual registration not allowed for this event",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}
	if err := validateParticipants(req.Data, event); err != nil {
		response := Response{
			Status: "error",
			Error:  err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if user.Registrations == nil {
		user.Registrations = make(map[string][]db.Participant)
	}

	participants := make([]db.Participant, len(req.Data))
	for i, p := range req.Data {
		participants[i] = db.Participant{
			Name:  p.Name,
			Email: p.Email,
			Class: p.Class,
			Phone: p.Phone,
		}
	}

	user.Registrations[req.EventID] = participants

	if err := globalDB.Update("users", email, user); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to update user registrations",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	registration := &db.Registration{
		EventID:   req.EventID,
		UserID:    user.ID,
		TeamName:  "",
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := globalDB.Create("registrations", registration); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to create registration",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	response := Response{
		Status:  "success",
		Message: "Registration submitted successfully",
		Data:    registration,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func validateParticipants(participants []Participant, event *db.Event) error {
	if len(participants) > event.Participants {
		return fmt.Errorf("maximum %d participants allowed", event.Participants)
	}
	if len(participants) == 0 {
		return fmt.Errorf("at least one participant required")
	}
	for i, participant := range participants {
		if err := validateParticipant(participant, event); err != nil {
			return fmt.Errorf("participant %d: %v", i+1, err)
		}
	}
	return nil
}

func validateParticipant(participant Participant, event *db.Event) error {
	participant.Name = strings.TrimSpace(strings.ToUpper(participant.Name))
	if participant.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !validateEmailFormat(participant.Email) {
		return fmt.Errorf("invalid email format: %s", participant.Email)
	}
	if participant.Class < 1 || participant.Class > 12 {
		return fmt.Errorf("class must be between 1 and 12")
	}
	var eligibility []int
	if err := json.Unmarshal([]byte(event.Eligibility), &eligibility); err != nil {
		return fmt.Errorf("invalid event eligibility format")
	}
	if len(eligibility) != 2 {
		return fmt.Errorf("invalid event eligibility format")
	}

	minClass := eligibility[0]
	maxClass := eligibility[1]
	if participant.Class < minClass || participant.Class > maxClass {
		return fmt.Errorf("class %d is not eligible for this event (eligible: %d-%d)", participant.Class, minClass, maxClass)
	}
	if len(participant.Phone) != 10 {
		return fmt.Errorf("phone number must be 10 digits")
	}
	if _, err := strconv.Atoi(participant.Phone); err != nil {
		return fmt.Errorf("phone number must contain only digits")
	}

	return nil
}

func validateEmailFormat(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func loadEventFromJSON(eventID string) (*db.Event, error) {
	b, err := ioutil.ReadFile("./frontend/data/events.json")
	if err != nil {
		return nil, err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}

	var eventMap map[string]string
	json.Unmarshal(top["events"], &eventMap)

	var participantsMap map[string]interface{}
	var eligibilityMap map[string]interface{}
	var individualMap map[string]interface{}

	json.Unmarshal(top["participants"], &participantsMap)
	json.Unmarshal(top["eligibility"], &eligibilityMap)
	json.Unmarshal(top["individual"], &individualMap)

	for name, image := range eventMap {
		if name == eventID || slugify(name) == eventID {
			evt := &db.Event{
				ID:    name,
				Name:  name,
				Image: image,
			}
			if p, ok := participantsMap[name]; ok {
				switch v := p.(type) {
				case float64:
					evt.Participants = int(v)
				case int:
					evt.Participants = v
				}
			} else {
				evt.Participants = 1
			}
			if e, ok := eligibilityMap[name]; ok {
				if bytes, err := json.Marshal(e); err == nil {
					evt.Eligibility = string(bytes)
				}
			}
			if ind, ok := individualMap[name]; ok {
				if b, err := json.Marshal(ind); err == nil {
					var flag bool
					if err := json.Unmarshal(b, &flag); err == nil {
						evt.IndependentRegistration = flag
					}
				}
			}
			return evt, nil
		}
	}
	return nil, fmt.Errorf("event not found in JSON")
}
