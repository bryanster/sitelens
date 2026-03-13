package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"sitelens/internal/db"
	"sitelens/internal/handlers"
	"sitelens/internal/llm"
	"sitelens/internal/lmstudio"
	"sitelens/internal/ollama"
)

//go:embed web/templates/*.html
var templateFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

func main() {
	cfg := LoadConfig()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	var categorizer llm.Categorizer
	switch strings.ToLower(cfg.LLMProvider) {
	case "ollama":
		model := cfg.LLMModel
		if model == "" {
			model = "llama3.2"
		}
		categorizer = ollama.New(cfg.LLMURL, model)
		log.Printf("Provider: Ollama | URL: %s | Model: %s", cfg.LLMURL, model)
	default: // "lmstudio"
		categorizer = lmstudio.New(cfg.LLMURL, cfg.LLMModel)
		log.Printf("Provider: LM Studio | URL: %s | Model: %s (empty = loaded model)", cfg.LLMURL, cfg.LLMModel)
	}

	if !categorizer.HealthCheck() {
		log.Printf("WARNING: LLM provider not reachable at %s — categorization will fail until it is running", cfg.LLMURL)
	}

	funcMap := template.FuncMap{
		"slugify": func(s string) string {
			s = strings.ToLower(s)
			s = strings.ReplaceAll(s, "/", "-")
			s = strings.ReplaceAll(s, " ", "-")
			return s
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "web/templates/*.html")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	h := handlers.New(database, categorizer, cfg.LLMModel, tmpl)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	staticSubFS, err := fs.Sub(staticFS, "web")
	if err != nil {
		log.Fatalf("sub static fs: %v", err)
	}
	r.Handle("/static/*", http.FileServer(http.FS(staticSubFS)))
	r.Get("/", h.Index)
	r.Get("/search", h.SearchPage)
	r.Post("/api/sites", h.AddSites)
	r.Get("/api/sites", h.ListSites)
	r.Delete("/api/sites/{id}", h.DeleteSite)
	r.Post("/api/sites/{id}/recategorize", h.Recategorize)
	r.Get("/api/export", h.Export)
	r.Get("/api/status", h.Status)
	r.Get("/api/search", h.SearchAPI)
	r.Get("/api/sites/batch", h.BatchSites)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("SiteLens running at http://localhost%s | DB: %s", addr, cfg.DBPath)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
