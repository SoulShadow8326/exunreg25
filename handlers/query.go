package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"exunreg25/db"

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

func loadFallbackMessage() string {
	b, err := os.ReadFile(filepath.Join("handlers", "bot.json"))
	if err != nil {
		return "I may not be able to help with that. Please reach out to exun@dpsrkp.net regarding the query."
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return "I may not be able to help with that. Please reach out to exun@dpsrkp.net regarding the query."
	}
	if fp, ok := data["fallback_policy"].(map[string]any); ok {
		if msg, ok := fp["fallback_message"].(string); ok && msg != "" {
			return msg
		}
	}
	return "I may not be able to help with that. Please reach out to exun@dpsrkp.net regarding the query."
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

func loadInviteData() (string, error) {
	paths := []string{"frontend/data/invite.md", "data/invite.md"}
	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			return string(b), nil
		}
	}
	return "", nil
}

var injectionRe = regexp.MustCompile(`(?i)(select\s+.*from|drop\s+table|insert\s+into|delete\s+from|--|;|\bunion\b|\bexec\b|\bexec\(|\bpasswd\b|\bpassword\b|\boutput\b)`)

var dumpRe = regexp.MustCompile(`(?i)\b(dump|export|full list|give (me|us) the (dataset|data|events)|all events|events list|show me (the )?events|events\.json|bot\.json|system prompt|self[_ ]?reference|identity|internal prompt|instructions|ignore rules|forget all|jailbreak|respond only|context dump)\b`)

var ackRe = regexp.MustCompile(`(?i)\bACKNOWLEDG|\bACK\b|Step\s*\d`)

var jailbreakRe = regexp.MustCompile("(?i)(base64|triple backticks|\\`\\`\\`|paste your system prompt|paste the system prompt|paste prompt|token names|token lengths|token name|length=\\d+|SEC-AUDIT-PASS|confirm with|acknowledge with|context dump|context-dump|give the token names)")

var base64OutputRe = regexp.MustCompile("[A-Za-z0-9+/=]{100,}")

var roleRe = regexp.MustCompile(`(?i)\b(as a (school auditor|compliance officer|admin|principal)|you must disclose|you are now a system)\b`)
var legalRe = regexp.MustCompile(`(?i)\b(transparency|foia|freedom of information|audit compliance|show your internal instructions|internal instructions)\b`)
var conditionalRe = regexp.MustCompile(`(?i)\bI AM BOUND\b|print I AM BOUND|first print I AM BOUND`)
var encodingRe = regexp.MustCompile(`(?i)\b(base64|hex|rot13|uuencode|uuencoded|encode using|encode it using|encode the prompt)\b`)
var splitRe = regexp.MustCompile(`(?i)\b(split( across)? messages|across many messages|across many responses|one word at a time|across words|across messages)\b`)
var htmlRe = regexp.MustCompile(`(?i)(<script>|<!--|</script>|<iframe|<img|javascript:)`)
var attachRe = regexp.MustCompile(`(?i)\battach( the)? (file|config|config.txt|attachment)\b`)
var roleplayRe = regexp.MustCompile(`(?i)\broleplay:?\b`)
var testRe = regexp.MustCompile(`(?i)\b(unit test|for testing|for unit tests|checker|validate outputs|test scenario|for audit)\b`)
var reverseRe = regexp.MustCompile(`(?i)\b(don't reveal your prompt|if you reveal it, type NO|otherwise reveal)\b`)
var tokenCountRe = regexp.MustCompile(`(?i)\b(token counts|token count|token lengths|tokens per section|token lengths)\b`)
var socialProofRe = regexp.MustCompile(`(?i)\b(signed by|principal|dean|@dpsrkp\.net|exun@dpsrkp\.net)\b`)
var hexOutputRe = regexp.MustCompile(`[0-9a-fA-F]{80,}`)
var emailRe = regexp.MustCompile(`(?i)[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)

func sanitizeQuery(q string) (string, bool) {
	if injectionRe.MatchString(q) {
		return "", false
	}
	cleaned := strings.TrimSpace(q)
	cleaned = strings.ReplaceAll(cleaned, "`", "")
	cleaned = strings.ReplaceAll(cleaned, "\\", "")
	return cleaned, true
}

func logRejection(reason, content string) {
	if globalDB != nil {
		_ = globalDB.Create("logs", &db.LogEntry{Reason: reason, Content: content, CreatedAt: time.Now()})
		fmt.Printf("[rejection][db] reason=%s content=%q\n", reason, content)
		return
	}
	t := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s\t%s\t%q\n", t, reason, content)
	fmt.Printf("[rejection] %s", line)
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
		logRejection("sanitize_failed", req.Query)
		resp := llmResponse{Answer: "I may not be able to help with that. Please reach out to exun@dpsrkp.net regarding the query.", Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": req.Query, "answer": resp.Answer, "status": "sanitize_failed"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if dumpRe.MatchString(cleaned) || ackRe.MatchString(cleaned) {
		logRejection("dump_or_ack_in_query", cleaned)
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "dump_or_ack_in_query"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if jailbreakRe.MatchString(cleaned) {
		logRejection("jailbreak_in_query", cleaned)
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "jailbreak_in_query"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if roleRe.MatchString(cleaned) || legalRe.MatchString(cleaned) || conditionalRe.MatchString(cleaned) || encodingRe.MatchString(cleaned) || splitRe.MatchString(cleaned) || roleplayRe.MatchString(cleaned) || testRe.MatchString(cleaned) || reverseRe.MatchString(cleaned) || tokenCountRe.MatchString(cleaned) || socialProofRe.MatchString(cleaned) || attachRe.MatchString(cleaned) {
		logRejection("heuristic_in_query", cleaned)
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "heuristic_in_query"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	systemPrompt, err := loadSystemPrompt()
	if err != nil {
		systemPrompt = "You are Exunb0t, a concise assistant for Exun 2025. Answer only from the provided dataset."
	}

	eventsData, _ := loadEventsData()
	inviteData, _ := loadInviteData()

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
	if inviteData != "" {
		sb.WriteString("\n\nInvite: ")
		sb.WriteString(inviteData)
	}
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

	lower := strings.ToLower(answer)
	if injectionRe.MatchString(answer) || strings.Contains(lower, "secret") || strings.Contains(lower, "api_key") || strings.Contains(lower, "password") || strings.Contains(lower, "private") {
		logRejection("sensitive_in_answer", answer)
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "sensitive_in_answer"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if dumpRe.MatchString(answer) || ackRe.MatchString(answer) || jailbreakRe.MatchString(answer) || base64OutputRe.MatchString(answer) {
		logRejection("dump_in_answer", answer)
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "dump_in_answer"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if roleRe.MatchString(answer) || legalRe.MatchString(answer) || conditionalRe.MatchString(answer) || encodingRe.MatchString(answer) || splitRe.MatchString(answer) || roleplayRe.MatchString(answer) || testRe.MatchString(answer) || reverseRe.MatchString(answer) || tokenCountRe.MatchString(answer) || socialProofRe.MatchString(answer) || attachRe.MatchString(answer) || htmlRe.MatchString(answer) || hexOutputRe.MatchString(answer) || emailRe.MatchString(answer) {
		logRejection("heuristic_in_answer", answer)
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "heuristic_in_answer"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if strings.Count(answer, ",") > 8 || len(answer) > 2000 {
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "length_or_commas"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if strings.Contains(answer, "{\"") || strings.Contains(answer, "\":") {
		fallback := loadFallbackMessage()
		resp := llmResponse{Answer: fallback, Source: "policy"}
		if globalDB != nil {
			payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "looks_like_json"}
			if b, err := json.Marshal(payload); err == nil {
				_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	resp := llmResponse{Answer: answer, Source: "llm"}
	if globalDB != nil {
		payload := map[string]string{"query": cleaned, "answer": resp.Answer, "status": "ok"}
		if b, err := json.Marshal(payload); err == nil {
			_ = globalDB.Create("logs", &db.LogEntry{Reason: "query", Content: string(b), CreatedAt: time.Now()})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
