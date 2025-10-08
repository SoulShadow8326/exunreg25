package handlers

import (
	"exunreg25/db"
	"exunreg25/templates"
	"net/http"
)

func LogPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := ""
	if c, err := r.Cookie("email"); err == nil {
		email = c.Value
	}
	if !IsAdminEmail(email) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	data := templates.TemplateData{
		IsAuthenticated: true,
		IsAdmin:         true,
		PageTitle:       "Logs | Exun 2025",
		CurrentPath:     "/log",
	}

	if globalDB != nil {
		if all, err := globalDB.GetAll("logs"); err == nil {
			for _, item := range all {
				if le, ok := item.(*db.LogEntry); ok {
					data.Logs = append(data.Logs, templates.LogItem{ID: le.ID, Reason: le.Reason, Content: le.Content, CreatedAt: le.CreatedAt})
				}
			}
		}
	}

	_ = templates.RenderTemplate(w, "log", data)
}
