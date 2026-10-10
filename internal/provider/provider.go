package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Input represents the error text to be explained.
type Input struct {
	ErrorText string
}

// Answer represents the explanation from the provider.
type Answer struct {
	Reason string
	Fix    string
}

// Provider is the interface for LLM providers.
type Provider interface {
	Ask(ctx context.Context, input Input) (Answer, error)
}

// postJSON sends a JSON POST request and returns the response body.
func postJSON(ctx context.Context, client *http.Client, url, apiKey string, payload any) ([]byte, error) {
	headers := map[string]string{}
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	return postJSONWithHeaders(ctx, client, url, headers, payload)
}

// postJSONWithHeaders sends a JSON POST request with custom headers and
// returns the response body. The URL must never contain secrets: net/http
// errors echo the URL, so anything in it can surface in error messages.
func postJSONWithHeaders(ctx context.Context, client *http.Client, url string, headers map[string]string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, URL: url}
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}

// HTTPError represents an HTTP error from the provider.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	switch e.StatusCode {
	case 400:
		return "Geçersiz istek (400)"
	case 401, 403:
		return "API anahtarı geçersiz veya yetkisiz (401/403)"
	case 404:
		return "Model bulunamadı (404)"
	case 429:
		return "Rate limit aşıldı (429)"
	default:
		if e.StatusCode >= 500 {
			return "Sunucu hatası (" + fmt.Sprint(e.StatusCode) + ")"
		}
		return "HTTP hatası: " + fmt.Sprint(e.StatusCode)
	}
}

// extractAnswer parses the provider response and extracts reason and fix.
func extractAnswer(body string) (Answer, error) {
	body = strings.TrimSpace(body)

	// Try strict JSON parsing first
	var parsed map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err == nil {
		reason := findKeyAny(parsed, "reason", "neden", "cause")
		fix := findKeyAny(parsed, "fix", "çözüm", "cozum", "command", "solution")
		if reason == "" {
			reason = "Açıklama alınamadı."
		}
		return Answer{Reason: reason, Fix: cleanFix(fix)}, nil
	}

	// Try balanced brace scanner for JSON object
	if jsonObj := extractFirstJSONObject(body); jsonObj != "" {
		if err := json.Unmarshal([]byte(jsonObj), &parsed); err == nil {
			reason := findKeyAny(parsed, "reason", "neden", "cause")
			fix := findKeyAny(parsed, "fix", "çözüm", "cozum", "command", "solution")
			if reason == "" {
				reason = "Açıklama alınamadı."
			}
			return Answer{Reason: reason, Fix: cleanFix(fix)}, nil
		}
	}

	// Fallback 1: Plain text with labeled sections
	if reason, fix := extractLabeled(body); reason != "" || fix != "" {
		if reason == "" {
			reason = "Açıklama alınamadı."
		}
		return Answer{Reason: reason, Fix: cleanFix(fix)}, nil
	}

	// Fallback 2: Fenced code block
	if reason, fix := extractFencedCodeBlock(body); reason != "" || fix != "" {
		if reason == "" {
			reason = "Açıklama alınamadı."
		}
		return Answer{Reason: reason, Fix: cleanFix(fix)}, nil
	}

	// Fallback 3: First line = reason, rest = fix
	lines := strings.Split(body, "\n")
	if len(lines) > 0 {
		reason := strings.TrimSpace(lines[0])
		fix := strings.TrimSpace(strings.Join(lines[1:], "\n"))
		if reason == "" {
			reason = "Açıklama alınamadı."
		}
		return Answer{Reason: reason, Fix: cleanFix(fix)}, nil
	}

	return Answer{Reason: "Açıklama alınamadı.", Fix: ""}, nil
}

func findKeyAny(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[strings.ToLower(k)]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func findKey(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[strings.ToLower(k)]; ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// extractFirstJSONObject finds the first balanced {...} object in the string.
func extractFirstJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' && !escape {
			inString = !inString
			continue
		}
		if !inString {
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					return s[start : i+1]
				}
			}
		}
	}
	return ""
}

// extractLabeled looks for "Label: value" patterns.
func extractLabeled(s string) (reason, fix string) {
	reasonLabels := []string{"neden:", "reason:", "cause:"}
	fixLabels := []string{"çözüm:", "cozum:", "fix:", "command:", "solution:"}

	lower := strings.ToLower(s)
	for _, rl := range reasonLabels {
		if idx := strings.Index(lower, rl); idx != -1 {
			after := strings.TrimSpace(s[idx+len(rl):])
			// Find next label or end
			end := len(after)
			for _, fl := range fixLabels {
				if fidx := strings.Index(strings.ToLower(after), fl); fidx != -1 && fidx < end {
					end = fidx
				}
			}
			reason = strings.TrimSpace(after[:end])
			// Extract fix if present
			for _, fl := range fixLabels {
				if fidx := strings.Index(lower, fl); fidx != -1 {
					fix = strings.TrimSpace(s[fidx+len(fl):])
					return reason, fix
				}
			}
			return reason, ""
		}
	}
	return "", ""
}

// extractFencedCodeBlock extracts content from ```...``` blocks.
func extractFencedCodeBlock(s string) (reason, fix string) {
	re := regexp.MustCompile("(?s)```(?:\\w+)?\\s*(.*?)\\s*```")
	matches := re.FindAllStringSubmatch(s, -1)
	if len(matches) > 0 {
		fix = strings.TrimSpace(matches[len(matches)-1][1])
		// Reason is everything before the last code block
		lastIdx := strings.LastIndex(s, matches[len(matches)-1][0])
		if lastIdx > 0 {
			reason = strings.TrimSpace(s[:lastIdx])
		}
		return reason, fix
	}
	return "", ""
}

// cleanFix removes markdown fences, backticks, and $ prompts from fix.
func cleanFix(fix string) string {
	// Remove ```...``` fences
	fix = regexp.MustCompile("(?s)```(?:\\w+)?\\s*(.*?)\\s*```").ReplaceAllString(fix, "$1")
	// Remove inline backticks
	fix = strings.ReplaceAll(fix, "`", "")
	// Remove $ prompts at line starts
	lines := strings.Split(fix, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "$"))
	}
	fix = strings.Join(lines, "\n")
	return strings.TrimSpace(fix)
}

// Common system prompt for all providers.
const systemPromptTR = `Sen bir terminal hata açıklama asistanısın. Kullanıcının hata mesajını alıp, sadece şu JSON formatında yanıt ver:
{"reason": "<tek cümlelik neden>", "fix": "<kopyalanabilir düzeltme komutu(ları)>"}

Kurallar:
- reason: Tek cümle, Türkçe, nokta ile biten.
- fix: Sadece komut(lar), açıklama yok, markdown/kod bloğu yok, $ işareti yok.
- JSON dışında HIÇBİR ŞEY yazma.`

const systemPromptEN = `You are a terminal error explanation assistant. Given an error message, respond ONLY in this JSON format:
{"reason": "<one-sentence cause>", "fix": "<copy-pasteable fix command(s)>"}

Rules:
- reason: One sentence, English, ends with period.
- fix: Only command(s), no explanation, no markdown/code fences, no $ prompt.
- Write NOTHING except the JSON.`
