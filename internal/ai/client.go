// Package ai provides an OpenAI-compatible request generator. The transport and
// cancellation pattern follows Tomori's ai.go; output uses Sento's .http grammar.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/algonacci/sento/internal/httpfile"
)

type Config struct {
	BaseURL string
	Model   string
	APIKey  string
}

const SystemPrompt = `Generate a Sento .http file from the user's API description. Return only the file, without Markdown fences or commentary.
Grammar:
@baseUrl = https://api.example.com

### Human-readable request title
# @name unique-request-id
GET {{baseUrl}}/path
Accept: application/json
Authorization: Bearer {{env.API_TOKEN}}

### Create an item
# @name create-item
POST {{baseUrl}}/items
Content-Type: application/json

{"name":"example"}

Use ### between requests, unique stable IDs containing letters, digits, hyphens, underscores or dots. Variables are file-global and declared once. Use {{env.NAME}} for runtime environment variables and secrets. Supported methods: GET POST PUT PATCH DELETE HEAD OPTIONS. Bodies are literal text after a blank line. No scripts, file directives, response chaining, or dynamic variables. Do not invent credentials. Prefer declared variables for unknown hosts and values. You generate definitions; you do not execute requests.`

func Generate(ctx context.Context, cfg Config, prompt string) (string, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("AI base URL must be HTTP(S), without credentials, query or fragment")
	}
	if strings.TrimSpace(cfg.Model) == "" || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("AI model and prompt are required")
	}
	if len(prompt) > 32<<10 {
		return "", fmt.Errorf("AI prompt exceeds 32 KiB")
	}
	payload, err := json.Marshal(map[string]any{
		"model": cfg.Model, "stream": false,
		"messages": []map[string]string{{"role": "system", "content": SystemPrompt}, {"role": "user", "content": prompt}},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	client := &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI provider returned HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 1<<20 {
		return "", fmt.Errorf("AI response exceeds 1 MiB")
	}
	var result struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("invalid AI response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI provider returned no choices")
	}
	text := strings.TrimSpace(result.Choices[0].Message.Content)
	if strings.HasPrefix(text, "```http\n") && strings.HasSuffix(text, "\n```") {
		text = strings.TrimSuffix(strings.TrimPrefix(text, "```http\n"), "\n```")
	}
	if _, err := httpfile.Parse(text); err != nil {
		return "", fmt.Errorf("AI generated invalid .http: %w", err)
	}
	return text + "\n", nil
}
