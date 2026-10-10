package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// geminiProvider implements Provider for native Gemini API.
type geminiProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// Ask sends the error to the Gemini API and returns the explanation.
func (p *geminiProvider) Ask(ctx context.Context, input Input) (Answer, error) {
	url := strings.TrimRight(p.baseURL, "/")
	if !strings.HasSuffix(url, ":generateContent") {
		url += ":generateContent"
	}

	payload := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": systemPromptTR}},
		},
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": input.ErrorText}}},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"temperature":      0.1,
		},
	}

	// The API key travels in the x-goog-api-key header, never in the URL:
	// net/http errors echo the URL, so a ?key= parameter would leak it.
	headers := map[string]string{}
	if p.apiKey != "" {
		headers["x-goog-api-key"] = p.apiKey
	}
	body, err := postJSONWithHeaders(ctx, p.client, url, headers, payload)
	if err != nil {
		return Answer{}, err
	}

	// Parse Gemini response format
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Answer{}, err
	}

	if resp.Error != nil {
		return Answer{}, &HTTPError{StatusCode: resp.Error.Code, URL: url}
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return Answer{}, errors.New("empty response from Gemini")
	}

	// Gemini returns JSON in the text field when responseMimeType is application/json
	content := resp.Candidates[0].Content.Parts[0].Text
	ans, err := extractAnswer(content)
	if err != nil {
		return Answer{}, err
	}
	return ans, nil
}

// NewGeminiProvider creates a new native Gemini provider.
func NewGeminiProvider(baseURL, apiKey, model string, timeout time.Duration) Provider {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta/models/" + model
	} else if !strings.Contains(baseURL, "/models/") {
		baseURL = strings.TrimRight(baseURL, "/") + "/models/" + model
	}
	return &geminiProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

// formatGeminiError formats Gemini API errors nicely.
func formatGeminiError(e *HTTPError) string {
	if e.StatusCode == 400 {
		return "Geçersiz istek: model adı yanlış olabilir"
	}
	if e.StatusCode == 401 || e.StatusCode == 403 {
		return "API anahtarı geçersiz veya yetkisiz"
	}
	if e.StatusCode == 404 {
		return "Model bulunamadı"
	}
	if e.StatusCode == 429 {
		return "Rate limit aşıldı, lütfen bekleyin"
	}
	if e.StatusCode >= 500 {
		return "Sunucu hatası (" + fmt.Sprint(e.StatusCode) + ")"
	}
	return "HTTP hatası: " + fmt.Sprint(e.StatusCode)
}
