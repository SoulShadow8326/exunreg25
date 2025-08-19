package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"exunreg25/config"
	"exunreg25/db"
	"exunreg25/handlers"
	"exunreg25/mail"
	"exunreg25/middleware"
	"exunreg25/routes"
	"exunreg25/templates"
)

func init() {
	loadEnv()
}
func loadEnv() {
	envFile, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	lines := strings.Split(string(envFile), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}
func main() {
	port := flag.String("port", "8080", "HTTP server port")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	if cfg.AuthSalt == "" {
		log.Fatal("AUTH_SALT environment variable is required")
	}

	database, err := db.NewConnection(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	if err := database.InitTables(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	authConfig := &handlers.AuthConfig{
		Salt:         cfg.AuthSalt,
		CookieSecure: cfg.CookieSecure,
	}

	emailConfig := &mail.EmailConfig{
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUsername: cfg.SMTPUsername,
		SMTPPassword: cfg.SMTPPassword,
		FromEmail:    cfg.FromEmail,
		FromName:     cfg.FromName,
	}

	emailService := mail.NewEmailService(emailConfig)
	authHandler := handlers.NewAuthHandler(database, authConfig, emailService)
	adminHandler := handlers.NewAdminHandler(database)

	handlers.SetGlobalAuthHandler(authHandler)
	handlers.SetGlobalAdminHandler(adminHandler)
	handlers.SetGlobalDB(database)

	if err := templates.InitTemplates(); err != nil {
		log.Fatal("Failed to initialize templates:", err)
	}

	handler := routes.SetupRoutes()
	wrappedHandler := middleware.CORS(middleware.Logger(handler))

	server := &http.Server{
		Addr:    ":" + *port,
		Handler: wrappedHandler,
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Server running on port %s", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
