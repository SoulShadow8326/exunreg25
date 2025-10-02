package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/genai"
)

type llmRequest struct {
	Query string `json:"query"`
}

type llmResponse struct {
	Answer string `json:"answer"`
	Source string `json:"source"`
}

func loadSystemPrompt() (string, error) {
	b, err := os.ReadFile(filepath.Join("handlers", "bot.json"))
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return "", err
	}
	if identity, ok := data["identity"].(map[string]interface{}); ok {
		if sr, ok := identity["self_reference"].(string); ok {
			return sr, nil
		}
	}
	return string(b), nil
}

func loadEventsData() (string, error) {
	paths := []string{"frontend/data/events.json", "data/events.json"}
	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			return string(b), nil
		}
	}
	return "[]", nil
}

var injectionRe = regexp.MustCompile(`(?i)(select\s+.*from|drop\s+table|insert\s+into|delete\s+from|--|;|\bunion\b|\bexec\b|\bexec\(|\bpasswd\b|\bpassword\b|\boutput\b)`)

func sanitizeQuery(q string) (string, bool) {
	if injectionRe.MatchString(q) {
		return "", false
	}
	cleaned := strings.TrimSpace(q)
	cleaned = strings.ReplaceAll(cleaned, "`", "")
	cleaned = strings.ReplaceAll(cleaned, "\\", "")
	return cleaned, true
}

func QueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req llmRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cleaned, ok := sanitizeQuery(req.Query)
	if !ok || cleaned == "" {
		resp := llmResponse{Answer: "I’m not sure about that. Please reach out to exun@dpsrkp.net for further assistance.", Source: "policy"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	systemPrompt, err := loadSystemPrompt()
	if err != nil {
		systemPrompt = "You are Exunb0t, a concise assistant for Exun 2025. Answer only from the provided dataset."
	}

	eventsData, _ := loadEventsData()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		http.Error(w, "server misconfigured", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()

	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\nDataset: ")
	sb.WriteString(eventsData)
	sb.WriteString("\n\nUser query: ")
	sb.WriteString(cleaned)

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		http.Error(w, "llm error", http.StatusInternalServerError)
		return
	}

	result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(sb.String()), nil)
	if err != nil {
		http.Error(w, "llm error", http.StatusInternalServerError)
		return
	}
	answer := strings.TrimSpace(result.Text())

	if injectionRe.MatchString(answer) || strings.Contains(strings.ToLower(answer), "secret") || strings.Contains(strings.ToLower(answer), "api_key") {
		resp := llmResponse{Answer: "I’m not sure about that. Please reach out to exun@dpsrkp.net for further assistance.", Source: "policy"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp := llmResponse{Answer: answer, Source: "llm"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
