# SiteLens

A local web application that categorizes websites using a local LLM (Ollama or LM Studio). Submit URLs, and SiteLens will scrape each site and classify it into categories like Technology, Security, E-Commerce, News/Media, and more. Results are stored in a DuckDB database.

## Features

- Automatic website categorization via local LLMs (no data leaves your machine)
- Supports **Ollama** and **LM Studio** as LLM providers
- Async processing — submit URLs and results appear as they complete
- Search and filter sites by category
- Recategorize sites with a different model
- CSV export
- Categorization history tracking per site
- Dark-themed htmx UI with live polling

## Prerequisites

- **Go 1.22+**
- A local LLM provider (one of the following):
  - [LM Studio](https://lmstudio.ai/) — download and load any model, runs on port 1234 by default
  - [Ollama](https://ollama.com/) — install and pull a model (e.g. `ollama pull llama3.2`)

## Quick Start

```bash
# Clone the repo
git clone https://github.com/bryanster/sitelens.git
cd sitelens

# Run with LM Studio (default — uses whatever model is loaded on port 1234)
go run .

# Run with a specific LM Studio model
LLM_MODEL=mistral-7b go run .

# Run with Ollama
LLM_PROVIDER=ollama LLM_URL=http://localhost:11434 LLM_MODEL=llama3.2 go run .
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `LLM_PROVIDER` | `lmstudio` | LLM backend: `lmstudio` or `ollama` |
| `LLM_URL` | `http://localhost:1234` | Base URL of the LLM API |
| `LLM_MODEL` | `qwen/qwen3.5-9b` | Model name (empty string uses the loaded model in LM Studio) |
| `DB_PATH` | `./sitelens.db` | Path to the DuckDB database file |
| `PORT` | `8080` | HTTP server port |

## Categories

Sites are classified into one of the following categories:

- News/Media
- Social Media
- E-Commerce
- Technology
- Finance/Banking
- Entertainment
- Education
- Government
- Healthcare
- Security
- Hacking/Phishing
- Adult Content
- Logistics
- Energy
- Other

## API

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | Main UI |
| `GET` | `/search` | Search page |
| `POST` | `/api/sites` | Add URLs — body: `{"urls": ["https://example.com"]}` |
| `GET` | `/api/sites` | List all sites (optional `?category=` filter) |
| `DELETE` | `/api/sites/{id}` | Delete a site |
| `POST` | `/api/sites/{id}/recategorize` | Re-run categorization for a site |
| `GET` | `/api/export` | Download all sites as CSV |
| `GET` | `/api/status` | Check if any sites are pending: `{"pending": bool}` |
| `GET` | `/api/search?q=&category=` | Search sites by text and/or category |
| `GET` | `/api/sites/batch?ids=1,2,3` | Fetch specific sites by ID |
| `GET` | `/api/sites/{id}/history` | Get categorization history for a site |

## Project Structure

```
├── main.go                     # Server setup, routing, template init
├── config.go                   # Environment variable configuration
├── internal/
│   ├── db/db.go                # DuckDB database layer
│   ├── handlers/handlers.go    # HTTP handlers
│   ├── langchain/              # LangChain-based LLM integration
│   ├── llm/llm.go              # Categorizer interface & category definitions
│   └── scraper/scraper.go      # HTML scraping with goquery
└── web/
    ├── templates/index.html    # htmx frontend
    └── static/style.css        # Dark theme styles
```

## License

MIT
