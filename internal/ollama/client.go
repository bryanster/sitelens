package ollama

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

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

func (c *Client) Categorize(url, title, snippet string) (string, error) {
	prompt := fmt.Sprintf("URL: %s\nTitle: %s\nContent: %s", url, title, snippet)

	body, err := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: prompt,
		System: llm.SystemPrompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama HTTP %d", resp.StatusCode)
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	category := strings.TrimSpace(result.Response)
	if category == "" {
		category = "Other"
	}
	return category, nil
}

// HealthCheck returns true if the Ollama server is reachable.
func (c *Client) HealthCheck() bool {
	resp, err := httpClient.Get(c.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Ensure Client implements llm.Categorizer.
var _ llm.Categorizer = (*Client)(nil)
