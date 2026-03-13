package llm

// Categorizer is the common interface for any local LLM backend.
type Categorizer interface {
	Categorize(url, title, snippet string) (string, error)
	HealthCheck() bool
}

const SystemPrompt = `You are a website categorization engine. Given information about a website, respond with ONLY a single category name from the list below — nothing else.

Categories:
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
- hacking / phising
- Adult Content
- Other

Rules:
- Respond with exactly one category name as written above.
- Do not add punctuation, explanation, or extra words.
- If uncertain, use "Other".`
