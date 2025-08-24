package routes

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"exunreg25/handlers"
	"exunreg25/middleware"
	"exunreg25/templates"
	"mime"
	"net/http"
)

func slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func getTemplateData(r *http.Request) templates.TemplateData {
	data := templates.TemplateData{
		IsAuthenticated: middleware.IsAuthenticated(r),
		IsAdmin:         false,
		CurrentPath:     r.URL.Path,
	}

	if data.IsAuthenticated {
		if email := middleware.GetEmailFromCookie(r); email != "" {
			data.IsAdmin = handlers.IsAdminEmail(email)
		}
	}

	data.IsHome = (r.URL.Path == "/" || r.URL.Path == "/index")
	data.IsEvents = (r.URL.Path == "/events")
	data.IsBrochure = (r.URL.Path == "/brochure")

	return data
}

func escapeForJSLiteral(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "$", "\\$")
	return s
}

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/assets/", http.StripPrefix("/assets/", customFileServer("./frontend/assets/")))
	mux.Handle("/css/", http.StripPrefix("/css/", customFileServer("./frontend/css/")))
	mux.Handle("/js/", http.StripPrefix("/js/", customFileServer("./frontend/js/")))
	mux.Handle("/illustrations/", http.StripPrefix("/illustrations/", customFileServer("./frontend/illustrations/")))
	mux.Handle("/data/", http.StripPrefix("/data/", customFileServer("./frontend/data/")))
	mux.Handle("/fonts/", http.StripPrefix("/fonts/", customFileServer("./frontend/fonts/")))
	mux.Handle("/components/", http.StripPrefix("/components/", customFileServer("./frontend/components/")))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/", "/index":
			data := getTemplateData(r)
			data.PageTitle = "Exun 2025"
			if events, err := templates.LoadEventsFromJSON(); err == nil {
				data.Events = events
				cats := make(map[string]struct{})
				for _, e := range events {
					if e.Slug != "" {
						if strings.Contains(e.Name, "Build") {
							cats["build"] = struct{}{}
						}
						if strings.Contains(e.Name, "CubXL") {
							cats["cubing"] = struct{}{}
						}
					}
				}
				for k := range cats {
					data.Categories = append(data.Categories, templates.Category{Key: k, Name: strings.Title(k)})
				}
			}
			templates.RenderTemplate(w, "index", data)
			return
		case "/events":
			data := getTemplateData(r)
			data.PageTitle = "Events | Exun 2025"
			if events, err := templates.LoadEventsFromJSON(); err == nil {
				data.Events = events
				cats := make(map[string]struct{})
				for _, e := range events {
					if strings.Contains(e.Name, "Build") {
						cats["build"] = struct{}{}
					}
					if strings.Contains(e.Name, "CubXL") {
						cats["cubing"] = struct{}{}
					}
					if strings.Contains(e.Name, "DomainSquare") {
						cats["gaming"] = struct{}{}
					}
				}
				for k := range cats {
					data.Categories = append(data.Categories, templates.Category{Key: k, Name: strings.Title(k)})
				}
			}
			templates.RenderTemplate(w, "events", data)
			return
		case "/event-detail":
			data := getTemplateData(r)
			data.PageTitle = "Event | Exun 2025"
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Redirect(w, r, "/events", http.StatusSeeOther)
				return
			}
			if decoded, err := url.PathUnescape(id); err == nil && decoded != "" {
				id = decoded
			}
			slug := slugify(id)
			if event, err := templates.FindEventBySlug(slug); err == nil && event != nil {
				data.Event = event
				data.PageTitle = event.Name + " | Exun 2025"
			} else {
				http.Redirect(w, r, "/events", http.StatusSeeOther)
				return
			}
			templates.RenderTemplate(w, "event-detail", data)
			return
		case "/admin":
			data := getTemplateData(r)
			if !data.IsAuthenticated {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			if !data.IsAdmin {
				data.PageTitle = "Unauthorized | Exun 2025"
				templates.RenderTemplate(w, "404", data)
				return
			}
			data.PageTitle = "Admin Panel | Exun 2025"
			rec := httptest.NewRecorder()
			hreq := r.Clone(r.Context())
			handlers.GetAdminStats(rec, hreq)
			if rec.Code == http.StatusOK {
				var stats templates.AdminStats
				if err := json.Unmarshal(rec.Body.Bytes(), &stats); err == nil {
					data.Stats = &templates.AdminStats{
						TotalUsers:         stats.TotalUsers,
						TotalEvents:        stats.TotalEvents,
						TotalRegistrations: stats.TotalRegistrations,
						EventStats:         stats.EventStats,
					}
				}
			}
			templates.RenderTemplate(w, "admin", data)
			return
		case "/summary":
			data := getTemplateData(r)
			if !data.IsAuthenticated {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			data.PageTitle = "Summary | Exun 2025"
			rec := httptest.NewRecorder()
			hreq := r.Clone(r.Context())
			handlers.GetUserSummary(rec, hreq)
			if rec.Code == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err == nil {
					if dataMap, ok := resp["data"].(map[string]interface{}); ok {
						if ui, ok := dataMap["user_info"].(map[string]interface{}); ok {
							user := &templates.User{}
							if v, ok := ui["email"].(string); ok {
								user.Email = v
							}
							if v, ok := ui["fullname"].(string); ok {
								user.Fullname = v
							}
							if v, ok := ui["phone"].(string); ok {
								user.PhoneNumber = v
							}
							if v, ok := ui["institution_name"].(string); ok {
								user.InstitutionName = v
							}
							if v, ok := ui["principals_email"].(string); ok {
								user.PrincipalsEmail = v
							}
							if v, ok := ui["individual"].(bool); ok {
								user.Individual = v
							}
							data.User = user
						}

						summary := &templates.Summary{}
						if v, ok := dataMap["total_events_registered"].(float64); ok {
							summary.TotalRegistrations = int(v)
						}
						if v, ok := dataMap["total_participants"].(float64); ok {
							summary.ConfirmedRegistrations = int(v)
						}
						summary.PendingRegistrations = 0
						data.Summary = summary
					}
				}
			}
			templates.RenderTemplate(w, "summary", data)
			return
		case "/signup":
			if !middleware.IsAuthenticated(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			data := getTemplateData(r)
			data.PageTitle = "Complete Signup | Exun 2025"
			templates.RenderTemplate(w, "complete", data)
			return
		case "/complete":
			data := getTemplateData(r)
			data.PageTitle = "Complete Signup | Exun 2025"
			templates.RenderTemplate(w, "complete", data)
			return
		case "/brochure":
			data := getTemplateData(r)
			data.PageTitle = "Brochure | Exun 2025"
			if b, err := os.ReadFile("frontend/data/invite.md"); err == nil {
				data.BrochureMarkdown = string(b)
			} else if b2, err2 := os.ReadFile("data/invite.md"); err2 == nil {
				data.BrochureMarkdown = string(b2)
			}
			templates.RenderTemplate(w, "brochure", data)
			return
		case "/login":
			data := getTemplateData(r)
			data.PageTitle = "Login | Exun 2025"
			templates.RenderTemplate(w, "login", data)
			return
		}
		if strings.HasSuffix(path, ".html") {
			http.Redirect(w, r, strings.TrimSuffix(path, ".html"), http.StatusMovedPermanently)
			return
		}

		if path != "/" {
			raw := strings.TrimPrefix(path, "/")
			decoded, err := url.PathUnescape(raw)
			if err == nil && decoded != "" {
				if evs, err := handlers.GetAllEventsData(); err == nil {
					for _, ev := range evs {
						if ev.Name == decoded || slugify(ev.Name) == decoded || ev.ID == decoded {
							canonical := slugify(ev.Name)
							if decoded != canonical {
								http.Redirect(w, r, "/"+canonical, http.StatusMovedPermanently)
								return
							}
							data := getTemplateData(r)
							data.PageTitle = ev.Name + " | Exun 2025"
							if event, err := templates.FindEventBySlug(canonical); err == nil && event != nil {
								data.Event = event
							}
							templates.RenderTemplate(w, "event-detail", data)
							return
						}
					}
				}
			}
		}
		data := getTemplateData(r)
		data.PageTitle = "Page not found | Exun 2025"
		templates.RenderTemplate(w, "404", data)
	})

	mux.HandleFunc("/api/health", handlers.HealthCheck)

	mux.HandleFunc("/api/auth/send-otp", handlers.SendOTP)
	mux.HandleFunc("/api/auth/verify-otp", handlers.VerifyOTP)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)
	mux.HandleFunc("/api/auth/complete", handlers.CompleteSignup)

	mux.HandleFunc("/api/users/register", handlers.RegisterUser)
	mux.HandleFunc("/api/users/login", handlers.LoginUser)

	profileHandler := http.HandlerFunc(handlers.GetUserProfile)
	mux.Handle("/api/users/profile", middleware.AuthRequired(profileHandler))

	profileAuthHandler := http.HandlerFunc(handlers.GetProfile)
	mux.Handle("/api/auth/profile", middleware.AuthRequired(profileAuthHandler))

	mux.HandleFunc("/api/events", handlers.GetAllEvents)
	mux.HandleFunc("/api/events/", handlers.GetEvent)

	submitRegHandler := http.HandlerFunc(handlers.SubmitRegistrations)
	mux.Handle("/api/submit_registrations", middleware.AuthRequired(submitRegHandler))

	completeSignupPageHandler := http.HandlerFunc(handlers.CompleteSignupPage)
	mux.Handle("/api/complete", middleware.AuthRequired(completeSignupPageHandler))

	completeSignupAPIHandler := http.HandlerFunc(handlers.CompleteSignupAPI)
	mux.Handle("/api/complete_api", middleware.AuthRequired(completeSignupAPIHandler))

	userProfileHandler := http.HandlerFunc(handlers.GetUserProfileData)
	mux.Handle("/api/user/profile", middleware.AuthRequired(userProfileHandler))

	registrationHistoryHandler := http.HandlerFunc(handlers.GetUserRegistrationHistory)
	mux.Handle("/api/user/registration_history", middleware.AuthRequired(registrationHistoryHandler))

	summaryHandler := http.HandlerFunc(handlers.GetUserSummary)
	mux.Handle("/api/summary", middleware.AuthRequired(summaryHandler))

	adminStatsHandler := http.HandlerFunc(handlers.GetAdminStats)
	mux.Handle("/api/admin/stats", middleware.AuthRequired(adminStatsHandler))

	adminConfigHandler := http.HandlerFunc(handlers.GetAdminConfig)
	mux.Handle("/api/admin/config", middleware.AuthRequired(adminConfigHandler))

	adminEventHandler := http.HandlerFunc(handlers.GetAdminEvent)
	mux.Handle("/api/admin/events/", middleware.AuthRequired(adminEventHandler))

	adminUpdateEventHandler := http.HandlerFunc(handlers.UpdateEvent)
	mux.Handle("/api/admin/events", middleware.AuthRequired(adminUpdateEventHandler))

	adminDeleteEventHandler := http.HandlerFunc(handlers.DeleteEvent)
	mux.Handle("/api/admin/events/delete/", middleware.AuthRequired(adminDeleteEventHandler))

	adminUserDetailsHandler := http.HandlerFunc(handlers.GetUserDetails)
	mux.Handle("/api/admin/users", middleware.AuthRequired(adminUserDetailsHandler))

	adminEventRegistrationsHandler := http.HandlerFunc(handlers.GetEventRegistrations)
	mux.Handle("/api/admin/event-registrations", middleware.AuthRequired(adminEventRegistrationsHandler))

	adminExportHandler := http.HandlerFunc(handlers.ExportData)
	mux.Handle("/api/admin/export", middleware.AuthRequired(adminExportHandler))

	adminSendInviteHandler := http.HandlerFunc(handlers.SendInvite)
	mux.Handle("/api/admin/send-invite", middleware.AuthRequired(adminSendInviteHandler))
	adminImportEventsHandler := http.HandlerFunc(handlers.ImportEvents)
	mux.Handle("/api/admin/import_events", middleware.AuthRequired(adminImportEventsHandler))

	return mux
}
func customFileServer(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(r.URL.Path)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		default:
			if mt := mime.TypeByExtension(ext); mt != "" {
				w.Header().Set("Content-Type", mt)
			}
		}
		fs.ServeHTTP(w, r)
	})
}
func SetupServer() *http.Server {
	mux := SetupRoutes()

	handler := middleware.CORS(middleware.Logger(mux))

	return &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
}
