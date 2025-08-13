package mail

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type InviteEmailService struct {
	emailService *EmailService
}

type InviteEmailRequest struct {
	ToEmail       string `json:"to_email"`
	SchoolName    string `json:"school_name"`
	PrincipalName string `json:"principal_name"`
	CustomMessage string `json:"custom_message,omitempty"`
}

func NewInviteEmailService(emailService *EmailService) *InviteEmailService {
	return &InviteEmailService{
		emailService: emailService,
	}
}

func (ies *InviteEmailService) SendInviteEmail(req InviteEmailRequest) error {
	subject := "Exun 2025 Registration Invite"

	htmlContent, err := ies.generateInviteEmail(req)
	if err != nil {
		return fmt.Errorf("failed to generate invite email: %v", err)
	}

	return ies.emailService.SendEmail(req.ToEmail, subject, htmlContent)
}

func (ies *InviteEmailService) SendBulkInvites(emails []string, customMessage string) error {
	for _, email := range emails {
		req := InviteEmailRequest{
			ToEmail:       email,
			SchoolName:    "School",
			PrincipalName: "",
			CustomMessage: customMessage,
		}

		err := ies.SendInviteEmail(req)
		if err != nil {
			return fmt.Errorf("failed to send to %s: %v", email, err)
		}

		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (ies *InviteEmailService) generateInviteEmail(req InviteEmailRequest) (string, error) {
	templatePath := filepath.Join("mail", "invite.html")

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %v", err)
	}

	tmpl, err := template.New("invite").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	data := struct {
		SchoolName    string
		PrincipalName string
		CustomMessage string
		CurrentYear   int
		CurrentDate   string
	}{
		SchoolName:    req.SchoolName,
		PrincipalName: req.PrincipalName,
		CustomMessage: req.CustomMessage,
		CurrentYear:   time.Now().Year(),
		CurrentDate:   time.Now().Format("January 2, 2006"),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}

func (ies *InviteEmailService) SendReminderEmail(email, schoolName string) error {
	subject := "Reminder: Exun 2024 Registration Deadline Approaching"

	htmlContent, err := ies.generateReminderEmail(schoolName)
	if err != nil {
		return fmt.Errorf("failed to generate reminder email: %v", err)
	}

	return ies.emailService.SendEmail(email, subject, htmlContent)
}

func (ies *InviteEmailService) generateReminderEmail(schoolName string) (string, error) {
	templatePath := filepath.Join("mail", "reminder.html")

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %v", err)
	}

	tmpl, err := template.New("reminder").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	data := struct {
		SchoolName  string
		CurrentYear int
	}{
		SchoolName:  schoolName,
		CurrentYear: time.Now().Year(),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}

func (ies *InviteEmailService) SendWelcomeEmail(email, schoolName string) error {
	subject := "Welcome to Exun 2024 - Registration Confirmed"

	htmlContent, err := ies.generateWelcomeEmail(schoolName)
	if err != nil {
		return fmt.Errorf("failed to generate welcome email: %v", err)
	}

	return ies.emailService.SendEmail(email, subject, htmlContent)
}

func (ies *InviteEmailService) generateWelcomeEmail(schoolName string) (string, error) {
	templatePath := filepath.Join("mail", "welcome.html")

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %v", err)
	}

	tmpl, err := template.New("welcome").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	data := struct {
		SchoolName  string
		CurrentYear int
	}{
		SchoolName:  schoolName,
		CurrentYear: time.Now().Year(),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}