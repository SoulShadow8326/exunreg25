package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DBPath       string
	Port         string
	AuthSalt     string
	CookieSecure bool
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

func Load() (*Config, error) {
	loadEnvFile()

	authSalt := getEnv("AUTH_SALT", "")
	if authSalt == "" {
		return nil, fmt.Errorf("AUTH_SALT is required")
	}

	config := &Config{
		DBPath:       getEnv("DB_PATH", "./data/exunreg25.db"),
		Port:         getEnv("PORT", "8080"),
		AuthSalt:     authSalt,
		CookieSecure: getEnvBool("COOKIE_SECURE", true),
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", ""),
		FromName:     getEnv("FROM_NAME", ""),
	}

	return config, nil
}

func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}
