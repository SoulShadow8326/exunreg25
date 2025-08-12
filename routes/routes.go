package routes

import (
	"net/http"

	"exunreg25/handlers"
	"exunreg25/middleware"
)

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", handlers.HealthCheck)

	mux.HandleFunc("/api/auth/send-otp", handlers.SendOTP)
	mux.HandleFunc("/api/auth/verify-otp", handlers.VerifyOTP)
	mux.HandleFunc("/api/auth/logout", handlers.Logout)
	mux.HandleFunc("/api/auth/complete-signup", handlers.CompleteSignup)

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
	mux.Handle("/api/complete_signup", middleware.AuthRequired(completeSignupPageHandler))

	completeSignupAPIHandler := http.HandlerFunc(handlers.CompleteSignupAPI)
	mux.Handle("/api/complete_signup_api", middleware.AuthRequired(completeSignupAPIHandler))

	userProfileHandler := http.HandlerFunc(handlers.GetUserProfileData)
	mux.Handle("/api/user/profile", middleware.AuthRequired(userProfileHandler))

	registrationHistoryHandler := http.HandlerFunc(handlers.GetUserRegistrationHistory)
	mux.Handle("/api/user/registration_history", middleware.AuthRequired(registrationHistoryHandler))

	return mux
}

func SetupServer() *http.Server {
	mux := SetupRoutes()

	handler := middleware.CORS(middleware.Logger(mux))

	return &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}
}