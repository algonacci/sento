package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateUsesConfiguredProviderAndValidatesOutput(t *testing.T) {
	for _, answer := range []string{"### Read\n# @name read\nGET https://example.com", "```http\nGET https://example.com\n```", "here is some prose", "GET https://example.com\n\n> {% script %}"} {
		t.Run(answer, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/custom/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("request: %s %v", r.URL.Path, r.Header)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["model"] != "local-model" || body["tools"] != nil {
					t.Errorf("payload: %v", body)
				}
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": answer}}}})
			}))
			defer srv.Close()
			text, err := Generate(context.Background(), Config{BaseURL: srv.URL + "/custom/v1", Model: "local-model", APIKey: "test-key"}, "Describe an API")
			valid := strings.HasPrefix(answer, "###") || strings.HasPrefix(answer, "```http")
			if (err == nil) != valid {
				t.Fatalf("output %q, err %v", text, err)
			}
		})
	}
}

func TestProviderErrorsDoNotEchoResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "private-token", 401) }))
	defer srv.Close()
	_, err := Generate(context.Background(), Config{BaseURL: srv.URL, Model: "test"}, "test")
	if err == nil || strings.Contains(err.Error(), "private-token") {
		t.Fatalf("error: %v", err)
	}
}
