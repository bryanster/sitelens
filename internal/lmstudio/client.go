// Package lmstudio implements the llm.Categorizer interface using LM Studio's
// OpenAI-compatible REST API (POST /v1/chat/completions).
package lmstudio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sitelens/internal/llm"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

type Client struct {
	baseURL string
	model   string
}

func New(baseURL, model string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), model: model}
}

// OpenAI-compatible request/response types.

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float32   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) Categorize(url, title, snippet string) (string, error) {
	userMsg := fmt.Sprintf("URL: %s\nTitle: %s\nContent: %s", url, title, snippet)

	body, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: llm.SystemPrompt},
			{Role: "user", Content: userMsg},
		},
		Temperature: 0,
		MaxTokens:   20,
		Stream:      false,
	})
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Post(c.baseURL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("lmstudio request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lmstudio HTTP %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "Other", nil
	}

	category := strings.TrimSpace(result.Choices[0].Message.Content)
	if category == "" {
		category = "Other"
	}
	return category, nil
}

// HealthCheck returns true if LM Studio's API server is reachable.
func (c *Client) HealthCheck() bool {
	resp, err := httpClient.Get(c.baseURL + "/v1/models")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Ensure Client implements llm.Categorizer.
var _ llm.Categorizer = (*Client)(nil)
