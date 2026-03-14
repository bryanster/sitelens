package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"html/template"
	"time"

	"github.com/go-chi/chi/v5"

	"sitelens/internal/db"
	"sitelens/internal/llm"
	"sitelens/internal/scraper"
)

type Handler struct {
	db    *db.DB
	llm   llm.Categorizer
	model string
	tmpl  *template.Template
}

func New(database *db.DB, categorizer llm.Categorizer, model string, tmpl *template.Template) *Handler {
	return &Handler{db: database, llm: categorizer, model: model, tmpl: tmpl}
}

// GET /
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	sites, err := h.db.ListSites("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cats, _ := h.db.Categories()
	hasPending, _ := h.db.HasPending()
	stats, _ := h.db.Stats()

	data := map[string]any{
		"Sites":      sites,
		"Categories": cats,
		"HasPending": hasPending,
		"Stats":      stats,
		"ActivePage": "categorize",
		"CategoryDescriptions": llm.CategoryDescriptions,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "index.html", data)
}

// POST /api/sites — body: {"urls": ["https://..."]}
func (h *Handler) AddSites(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var inserted []int64
	for _, rawURL := range req.URLs {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}
		id, err := h.db.InsertSite(rawURL)
		if err != nil {
			log.Printf("insert %s: %v", rawURL, err)
			continue
		}
		inserted = append(inserted, id)
		go h.process(id, rawURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"inserted": len(inserted), "ids": inserted})
}

// GET /api/sites
func (h *Handler) ListSites(w http.ResponseWriter, r *http.Request) {
	cat := r.URL.Query().Get("category")
	sites, err := h.db.ListSites(cat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hasPending, _ := h.db.HasPending()
	cats, _ := h.db.Categories()

	// If htmx request, return partial HTML table rows
	if r.Header.Get("HX-Request") == "true" {
		data := map[string]any{
			"Sites":      sites,
			"Categories": cats,
			"HasPending": hasPending,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.tmpl.ExecuteTemplate(w, "table-body", data)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sites)
}

// DELETE /api/sites/{id}
func (h *Handler) DeleteSite(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.db.DeleteSite(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/sites/{id}/recategorize
func (h *Handler) Recategorize(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	site, err := h.db.GetSite(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.db.SetStatus(id, "pending", "")
	go h.process(id, site.URL)
	w.WriteHeader(http.StatusAccepted)
}

// GET /api/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	sites, err := h.db.ListSites("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("sitelens-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "url", "title", "category", "status", "created_at"})
	for _, s := range sites {
		cw.Write([]string{
			strconv.FormatInt(s.ID, 10),
			s.URL,
			s.Title,
			s.Category,
			s.Status,
			s.CreatedAt.Format(time.RFC3339),
		})
	}
	cw.Flush()
}

// GET /api/sites/batch?ids=1,2,3 — fetch specific sites by ID (for session tracking)
func (h *Handler) BatchSites(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("ids")
	cat := r.URL.Query().Get("category")
	var ids []int64
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.tmpl.ExecuteTemplate(w, "table-body", map[string]any{"Sites": nil})
		return
	}

	sites, err := h.db.ListSitesByIDs(ids, cat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" || r.Header.Get("Accept") == "text/html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.tmpl.ExecuteTemplate(w, "table-body", map[string]any{"Sites": sites})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sites)
}

// GET /search — search page
func (h *Handler) SearchPage(w http.ResponseWriter, r *http.Request) {
	cats, _ := h.db.Categories()
	stats, _ := h.db.Stats()
	data := map[string]any{
		"Categories": cats,
		"Stats":      stats,
		"Sites":      []any{},
		"ActivePage": "search",
		"CategoryDescriptions": llm.CategoryDescriptions,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "search.html", data)
}

// GET /api/search?q=QUERY&category=CAT — returns search-results partial
func (h *Handler) SearchAPI(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	cat := r.URL.Query().Get("category")

	sites, err := h.db.SearchSites(query, cat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "search-results", map[string]any{
		"Sites": sites,
		"Query": query,
	})
}

// GET /api/status — returns stats and whether any rows are still pending
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	stats, _ := h.db.Stats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"pending":    stats.Pending,
		"total":      stats.Total,
		"done":       stats.Done,
		"categories": stats.Categories,
	})
}

// process runs scrape + categorize in a goroutine.
func (h *Handler) process(id int64, rawURL string) {
	h.db.SetStatus(id, "processing", "")

	page, err := scraper.Fetch(rawURL)
	if err != nil {
		log.Printf("[scraper] %s: %v", rawURL, err)
		h.db.UpdateSite(id, "", "", "", "", "error", err.Error())
		h.db.InsertHistory(id, "", "", "error", err.Error())
		return
	}

	category, err := h.llm.Categorize(rawURL, page.Title, page.Snippet)
	if err != nil {
		log.Printf("[ollama] %s: %v", rawURL, err)
		h.db.UpdateSite(id, page.Title, page.Snippet, "", "", "error", err.Error())
		h.db.InsertHistory(id, "", "", "error", err.Error())
		return
	}

	h.db.UpdateSite(id, page.Title, page.Snippet, category, h.model, "done", "")
	h.db.InsertHistory(id, category, h.model, "done", "")
	log.Printf("[done] %s → %s", rawURL, category)
}

// GET /api/sites/{id}/history
func (h *Handler) SiteHistory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	entries, err := h.db.GetHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "history-panel", map[string]any{
		"SiteID":  id,
		"Entries": entries,
	})
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}
