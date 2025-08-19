package templates

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var templates *template.Template

type TemplateData struct {
	IsAuthenticated bool
	IsAdmin         bool
	IsHome          bool
	IsEvents        bool
	User            *User
	Event           *Event
	Events          []Event
	Stats           *AdminStats
	PageTitle       string
	CurrentPath     string
}

type User struct {
	Email           string `json:"email"`
	FullName        string `json:"fullname"`
	PhoneNumber     string `json:"phone_number"`
	PrincipalsEmail string `json:"principals_email"`
	InstitutionName string `json:"institution_name"`
	Address         string `json:"address"`
	PrincipalsName  string `json:"principals_name"`
	Individual      string `json:"individual"`
}

type Event struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Description     string `json:"description"`
	EligibilityText string `json:"eligibility_text"`
	Mode            string `json:"mode"`
	Image           string `json:"image"`
	Participants    int    `json:"participants"`
	MaxParticipants int    `json:"max_participants"`
	IsRegistered    bool   `json:"is_registered"`
}

type AdminStats struct {
	TotalUsers         int                  `json:"total_users"`
	TotalEvents        int                  `json:"total_events"`
	TotalRegistrations int                  `json:"total_registrations"`
	EventStats         map[string]EventStat `json:"event_stats"`
}

type EventStat struct {
	Name         string `json:"name"`
	Participants int    `json:"participants"`
}

func InitTemplates() error {
	var err error

	templates, err = template.ParseGlob("frontend/*.html")
	if err != nil {
		return err
	}

	componentTemplates, err := template.ParseGlob("frontend/components/*.html")
	if err != nil {
		return err
	}

	for _, tmpl := range componentTemplates.Templates() {
		templates.AddParseTree(tmpl.Name(), tmpl.Tree)
	}

	return nil
}

func RenderTemplate(w http.ResponseWriter, templateName string, data TemplateData) error {
	if templates == nil {
		if err := InitTemplates(); err != nil {
			return err
		}
	}

	baseName := filepath.Base(templateName)
	if filepath.Ext(baseName) == "" {
		baseName = baseName + ".html"
	}

	return templates.ExecuteTemplate(w, baseName, data)
}

func LoadEventsFromJSON() ([]Event, error) {
	data, err := os.ReadFile("frontend/data/events.json")
	if err != nil {
		return nil, err
	}

	var eventsData struct {
		Events map[string]string `json:"events"`
	}

	if err := json.Unmarshal(data, &eventsData); err != nil {
		return nil, err
	}

	var events []Event
	for name, image := range eventsData.Events {
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		slug = strings.ReplaceAll(slug, ":", "")
		events = append(events, Event{
			ID:    slug,
			Name:  name,
			Slug:  slug,
			Image: "/illustrations/" + image,
			Mode:  "online", 
		})
	}

	return events, nil
}

func FindEventBySlug(slug string) (*Event, error) {
	events, err := LoadEventsFromJSON()
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if event.Slug == slug {
			return &event, nil
		}
	}

	return nil, nil 
}
