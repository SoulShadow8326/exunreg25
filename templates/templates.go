package templates

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var templates *template.Template

type TemplateData struct {
	IsAuthenticated  bool
	IsAdmin          bool
	IsHome           bool
	IsEvents         bool
	IsBrochure       bool
	IsQuery          bool
	Logs             []LogItem
	BrochureMarkdown string
	BrochureNavHTML  template.HTML
	BrochureTOC      string
	BrochureScrollTo string
	User             *User
	Event            *Event
	Events           []Event
	Categories       []Category
	Stats            *AdminStats
	Summary          *Summary
	PageTitle        string
	CurrentPath      string
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

type LogItem struct {
	ID        int       `json:"id"`
	Reason    string    `json:"reason"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Summary struct {
	TotalRegistrations     int `json:"total_registrations"`
	ConfirmedRegistrations int `json:"confirmed_registrations"`
	PendingRegistrations   int `json:"pending_registrations"`
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

	if _, err := templates.New("chat").ParseFiles("frontend/components/chat.html"); err != nil {
		return err
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
	paths := []string{"frontend/data/events.json", "data/events.json"}
	var b []byte
	var err error
	for _, p := range paths {
		if bb, e := ioutil.ReadFile(p); e == nil {
			b = bb
			break
		} else {
			err = e
		}
	}
	if b == nil {
		return nil, err
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	var events []Event
	for _, ev := range raw {
		id := ""
		name := ""
		img := ""
		mode := ""
		participants := 0
		descShort := ""
		descLong := ""
		points := 0
		eligibility := ""
		dates := ""
		if v, ok := ev["id"].(string); ok {
			id = v
		}
		if v, ok := ev["name"].(string); ok {
			name = v
		}
		if v, ok := ev["image"].(string); ok {
			img = v
		}
		if v, ok := ev["mode"].(string); ok {
			mode = v
		}
		if v, ok := ev["participants"].(float64); ok {
			participants = int(v)
		}
		if v, ok := ev["description_short"].(string); ok {
			descShort = v
		}
		if v, ok := ev["description_long"].(string); ok {
			descLong = v
		}
		if v, ok := ev["points"].(float64); ok {
			points = int(v)
		}
		if v, ok := ev["eligibility"].(string); ok {
			eligibility = v
		}
		if v, ok := ev["dates"].(string); ok {
			dates = v
		}
		if img != "" && !strings.HasPrefix(img, "/") {
			img = "/illustrations/" + img
		}
		events = append(events, Event{
			ID:               id,
			Name:             name,
			Slug:             id,
			Image:            img,
			Mode:             mode,
			Participants:     participants,
			MaxParticipants:  0,
			IsRegistered:     false,
			DescriptionShort: descShort,
			DescriptionLong:  descLong,
			EligibilityText:  eligibility,
			Points:           points,
			Individual:       false,
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
