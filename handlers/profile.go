package handlers

import (
	"encoding/json"
	"exunreg25/db"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type CompleteSignupRequest struct {
	Fullname        string `json:"fullname"`
	PhoneNumber     string `json:"phone_number"`
	PrincipalsEmail string `json:"principals_email"`
	Individual      string `json:"individual"`
	InstitutionName string `json:"institution_name"`
	Address         string `json:"address"`
	PrincipalsName  string `json:"principals_name"`
}

type UserProfile struct {
	ID              int                    `json:"id"`
	Email           string                 `json:"email"`
	Username        string                 `json:"username"`
	Fullname        string                 `json:"fullname"`
	PhoneNumber     string                 `json:"phone_number"`
	PrincipalsEmail string                 `json:"principals_email"`
	Individual      string                 `json:"individual"`
	InstitutionName string                 `json:"institution_name"`
	Address         string                 `json:"address"`
	PrincipalsName  string                 `json:"principals_name"`
	Registrations   map[string]interface{} `json:"registrations"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type RegistrationHistory struct {
	EventID   string        `json:"event_id"`
	EventName string        `json:"event_name"`
	Status    string        `json:"status"`
	Data      []Participant `json:"data"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func CompleteSignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	if user.Username != "" {
		response := Response{
			Status: "error",
			Error:  "Signup already completed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Status:  "success",
		Message: "Complete signup page",
		Data: map[string]interface{}{
			"email": email,
			"fields": []string{
				"fullname",
				"phone_number",
				"principals_email",
				"individual",
				"institution_name",
				"address",
				"principals_name",
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func CompleteSignupAPI(w http.ResponseWriter, r *http.Request) {
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
	if user.Username != "" {
		response := Response{
			Status: "error",
			Error:  "Signup already completed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	var req CompleteSignupRequest
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

	if err := validateCompleteSignupRequest(req); err != nil {
		response := Response{
			Status: "error",
			Error:  err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	user.Fullname = strings.TrimSpace(strings.ToUpper(req.Fullname))
	user.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	user.PrincipalsEmail = strings.TrimSpace(req.PrincipalsEmail)
	user.Individual = strings.TrimSpace(req.Individual)
	user.InstitutionName = strings.TrimSpace(strings.ToUpper(req.InstitutionName))
	user.Address = strings.TrimSpace(strings.ToUpper(req.Address))
	user.PrincipalsName = strings.TrimSpace(strings.ToUpper(req.PrincipalsName))
	user.UpdatedAt = time.Now()

	if err := globalDB.Update("users", email, user); err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to update user profile",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Status:  "success",
		Message: "Profile completed successfully",
		Data:    user,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetUserProfileData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	profile := &UserProfile{
		ID:              user.ID,
		Email:           user.Email,
		Username:        user.Username,
		Fullname:        user.Fullname,
		PhoneNumber:     user.PhoneNumber,
		PrincipalsEmail: user.PrincipalsEmail,
		Individual:      user.Individual,
		InstitutionName: user.InstitutionName,
		Address:         user.Address,
		PrincipalsName:  user.PrincipalsName,
		Registrations:   make(map[string]interface{}),
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}

	registrations, err := getUserRegistrations(user.ID)
	if err == nil {
		profile.Registrations = registrations
	}

	response := Response{
		Status: "success",
		Data:   profile,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetUserRegistrationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	history, err := getUserRegistrations(user.ID)
	if err != nil {
		response := Response{
			Status: "error",
			Error:  "Failed to fetch registration history",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Status: "success",
		Data:   history,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func validateCompleteSignupRequest(req CompleteSignupRequest) error {
	if strings.TrimSpace(req.Fullname) == "" {
		return fmt.Errorf("fullname is required")
	}
	if strings.TrimSpace(req.PhoneNumber) == "" {
		return fmt.Errorf("phone number is required")
	}
	if strings.TrimSpace(req.PrincipalsEmail) == "" {
		return fmt.Errorf("principal's email is required")
	}
	if strings.TrimSpace(req.Individual) == "" {
		return fmt.Errorf("individual field is required")
	}
	if strings.TrimSpace(req.InstitutionName) == "" {
		return fmt.Errorf("institution name is required")
	}
	if strings.TrimSpace(req.Address) == "" {
		return fmt.Errorf("address is required")
	}
	if strings.TrimSpace(req.PrincipalsName) == "" {
		return fmt.Errorf("principal's name is required")
	}

	if len(req.PhoneNumber) != 10 {
		return fmt.Errorf("phone number must be 10 digits")
	}

	if !validateEmailFormat(req.PrincipalsEmail) {
		return fmt.Errorf("invalid principal's email format")
	}

	return nil
}

func getUserRegistrations(userID int) (map[string]interface{}, error) {
	registrations, err := globalDB.GetAll("registrations")
	if err != nil {
		return nil, err
	}

	userRegistrations := make(map[string]interface{})
	for _, regData := range registrations {
		reg := regData.(*db.Registration)
		if reg.UserID == userID {
			eventData, err := globalDB.Get("events", reg.EventID)
			if err != nil {
				continue
			}
			event := eventData.(*db.Event)

			userRegistrations[reg.EventID] = map[string]interface{}{
				"event_id":   reg.EventID,
				"event_name": event.Name,
				"status":     reg.Status,
				"created_at": reg.CreatedAt,
				"updated_at": reg.UpdatedAt,
			}
		}
	}

	return userRegistrations, nil
}
