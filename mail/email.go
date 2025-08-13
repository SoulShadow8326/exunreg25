package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type MailSender interface {
	SendOTP(to, otp string) error
}

type EmailService struct {
	config EmailConfig
}

func NewEmailService(config *EmailConfig) *EmailService {
	return &EmailService{config: *config}
}

func (es *EmailService) SendOTP(to, otp string) error {
	subject := fmt.Sprintf("Exun Registration Authentication OTP - %s", otp)

	htmlBody, err := es.renderOTPTemplate(otp)
	if err != nil {
		return fmt.Errorf("failed to render email template: %v", err)
	}

	return es.sendEmail(to, subject, htmlBody)
}

func (es *EmailService) SendEmail(to, subject, htmlBody string) error {
	return es.sendEmail(to, subject, htmlBody)
}

func (es *EmailService) renderOTPTemplate(otp string) (string, error) {
	templatePath := filepath.Join("mail", "otp.html")

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %v", err)
	}

	tmpl, err := template.New("otp").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	data := struct{ OTP string }{OTP: otp}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}

func (es *EmailService) sendEmail(to, subject, htmlBody string) error {
	if es.config.SMTPUsername == "" || es.config.SMTPPassword == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	auth := smtp.PlainAuth("", es.config.SMTPUsername, es.config.SMTPPassword, es.config.SMTPHost)

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", es.config.FromName, es.config.FromEmail),
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	addr := fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort)
	if err := smtp.SendMail(addr, auth, es.config.FromEmail, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("OTP email sent successfully to %s", to)
	return nil
}