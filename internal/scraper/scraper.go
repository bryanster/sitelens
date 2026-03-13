package scraper

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

const (
	snippetMaxChars = 600
	userAgent       = "Mozilla/5.0 (compatible; SiteLens/1.0)"
)

type Page struct {
	URL     string
	Title   string
	Snippet string
}

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

func Fetch(rawURL string) (*Page, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())

	// Prefer meta description, then fall back to body text
	snippet, _ := doc.Find(`meta[name="description"]`).Attr("content")
	snippet = strings.TrimSpace(snippet)

	if snippet == "" {
		var buf strings.Builder
		doc.Find("body").Find("p, h1, h2, h3, article").Each(func(_ int, s *goquery.Selection) {
			t := strings.TrimSpace(s.Text())
			if t != "" && buf.Len() < snippetMaxChars {
				buf.WriteString(t)
				buf.WriteString(" ")
			}
		})
		snippet = strings.TrimSpace(buf.String())
	}

	snippet = truncate(snippet, snippetMaxChars)

	return &Page{
		URL:     rawURL,
		Title:   title,
		Snippet: snippet,
	}, nil
}

func truncate(s string, maxChars int) string {
	if utf8.RuneCountInString(s) <= maxChars {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxChars]) + "…"
}
