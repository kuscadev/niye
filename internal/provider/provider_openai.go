package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// openAIProvider implements Provider for OpenAI-compatible APIs.
type openAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// Ask sends the error to the OpenAI-compatible API and returns the explanation.
func (p *openAIProvider) Ask(ctx context.Context, input Input) (Answer, error) {
	url := strings.TrimRight(p.baseURL, "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}

	payload := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPromptTR},
			{"role": "user", "content": input.ErrorText},
		},
		"temperature": 0.1,
	}

	body, err := postJSON(ctx, p.client, url, p.apiKey, payload)
	if err != nil {
		return Answer{}, err
	}

	// Parse OpenAI response format
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Answer{}, err
	}

	if resp.Error != nil {
		return Answer{}, &HTTPError{StatusCode: 400, URL: url}
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return Answer{}, errors.New("empty response from provider")
	}

	return extractAnswer(resp.Choices[0].Message.Content)
}

// NewOpenAIProvider creates a new OpenAI-compatible provider.
func NewOpenAIProvider(baseURL, apiKey, model string, timeout time.Duration) Provider {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &openAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}
