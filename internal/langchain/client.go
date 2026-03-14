package langchain

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"sitelens/internal/llm"
)

// Client wraps any langchaingo llms.Model behind the llm.Categorizer interface.
type Client struct {
	model     llms.Model
	healthURL string // optional URL to ping for HealthCheck
}

func New(model llms.Model, healthURL string) *Client {
	return &Client{model: model, healthURL: healthURL}
}

func (c *Client) Categorize(url, title, snippet string) (string, error) {
	userMsg := fmt.Sprintf("URL: %s\nTitle: %s\nContent: %s", url, title, snippet)

	msgs := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, llm.SystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userMsg),
	}

	resp, err := c.model.GenerateContent(context.Background(), msgs)
	if err != nil {
		return "Other", fmt.Errorf("llm generate: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "Other", nil
	}

	category := strings.TrimSpace(resp.Choices[0].Content)
	if category == "" {
		return "Other", nil
	}
	return category, nil
}

func (c *Client) HealthCheck() bool {
	if c.healthURL == "" {
		return true
	}
	resp, err := http.Get(c.healthURL) //nolint:gosec
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

var _ llm.Categorizer = (*Client)(nil)
