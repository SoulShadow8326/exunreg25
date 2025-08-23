package templates

import (
	"encoding/json"
	"fmt"
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
	Categories      []Category
	Stats           *AdminStats
	Summary         *Summary
	PageTitle       string
	CurrentPath     string
}

type Category struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type User struct {
	Email           string `json:"email"`
	Fullname        string `json:"fullname"`
	PhoneNumber     string `json:"phone_number"`
	PrincipalsEmail string `json:"principals_email"`
	InstitutionName string `json:"institution_name"`
	Address         string `json:"address"`
	PrincipalsName  string `json:"principals_name"`
	Individual      bool   `json:"individual"`
	Registrations   map[string][]Participant
}

type Participant struct {
	Name  string `json:"name"`
	Class int    `json:"class"`
}

type Summary struct {
	TotalRegistrations     int `json:"total_registrations"`
	ConfirmedRegistrations int `json:"confirmed_registrations"`
	PendingRegistrations   int `json:"pending_registrations"`
	TeamEvents             int `json:"team_events"`
}

type Event struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	DescriptionShort string `json:"description_short"`
	DescriptionLong  string `json:"description_long"`
	EligibilityText  string `json:"eligibility_text"`
	Mode             string `json:"mode"`
	Image            string `json:"image"`
	Participants     int    `json:"participants"`
	MaxParticipants  int    `json:"max_participants"`
	IsRegistered     bool   `json:"is_registered"`
	Points           int    `json:"points"`
	Individual       bool   `json:"individual"`
	Dates            string `json:"dates"`
}

type AdminStats struct {
	TotalUsers         int                   `json:"total_users"`
	TotalEvents        int                   `json:"total_events"`
	TotalRegistrations int                   `json:"total_registrations"`
	EventStats         map[string]EventStats `json:"event_stats"`
}

type EventStats struct {
	EventName         string `json:"event_name"`
	TotalParticipants int    `json:"total_participants"`
	TotalTeams        int    `json:"total_teams"`
	Mode              string `json:"mode"`
	Eligibility       string `json:"eligibility"`
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

	var raw struct {
		Events  map[string]string `json:"events"`
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

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var events []Event
	for name, image := range raw.Events {
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		slug = strings.ReplaceAll(slug, ":", "")
		slug = strings.ReplaceAll(slug, "+", "plus")

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

		eligibilityText := ""
		if openAll {
			eligibilityText = "Open to all"
		} else if vals, ok := raw.Eligibility[name]; ok && len(vals) >= 2 {
			eligibilityText = fmt.Sprintf("Grades %d–%d", vals[0], vals[1])
		} else if len(raw.Default.Eligibility) >= 2 {
			eligibilityText = fmt.Sprintf("Grades %d–%d", raw.Default.Eligibility[0], raw.Default.Eligibility[1])
		}

		dates := raw.Default.Dates

		events = append(events, Event{
			ID:               slug,
			Name:             name,
			Slug:             slug,
			Image:            "/illustrations/" + image,
			Mode:             mode,
			Participants:     participants,
			MaxParticipants:  0,
			IsRegistered:     false,
			DescriptionShort: descShort,
			DescriptionLong:  descLong,
			EligibilityText:  eligibilityText,
			Points:           points,
			Individual:       individual,
			Dates:            dates,
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
