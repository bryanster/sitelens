package llm

import (
	"regexp"
	"strings"
)

// Categorizer is the common interface for any local LLM backend.
type Categorizer interface {
	Categorize(url, title, snippet string) (string, error)
	HealthCheck() bool
}

// CategoryDescriptions maps each category to a detailed description
var CategoryDescriptions = map[string]string{
	"News/Media": "News outlets, journalism platforms, blogs, magazines, podcasts, and media distribution sites. Includes outlets covering politics, sports, technology, business, and entertainment news.",
	"Social Media": "Social networking platforms, community forums, messaging apps, content sharing networks, and user-generated content sites. Examples: Facebook, Twitter, Instagram, Reddit, TikTok, Discord.",
	"E-Commerce": "Online retail stores, marketplaces, shopping platforms, product catalogs, and payment processing sites. Includes stores selling physical goods, digital products, and services.",
	"Technology": "Software companies, tech news, developer tools, SaaS platforms, cloud services, programming resources, IT infrastructure, and tech documentation.",
	"Finance/Banking": "Banks, financial institutions, investment platforms, cryptocurrency exchanges, trading platforms, accounting software, and financial advisories.",
	"Entertainment": "Streaming services, movie and music databases, video content platforms, gaming sites, entertainment news, and digital media consumption.",
	"Education": "Schools, universities, online learning platforms, educational content, e-learning courses, libraries, academic resources, and training programs.",
	"Government": "Government agencies, public administration sites, official state/federal/local resources, public records, and government services.",
	"Healthcare": "Hospitals, clinics, medical professionals, health information, telemedicine, pharmaceutical information, wellness resources, and patient portals.",
	"Security": "Cybersecurity companies, security tools, vulnerability databases, security research, threat intelligence, password managers, and security consulting.",
	"hacking / phising": "Sites used for malicious hacking activities, phishing campaigns, exploit distribution, malware hosting, credential theft, or other cyber attacks.",
	"Adult Content": "Adult entertainment, dating sites with explicit content, and other adult-oriented services.",
	"Logistics": "Shipping companies, delivery services, supply chain management, warehousing, package tracking, freight services, and logistics platforms.",
	"Energy": "Power companies, energy utilities, renewable energy providers, energy trading, oil and gas companies, and energy infrastructure.",
	"Other": "Websites that don't fit into any of the above categories or are unclear based on available information.",
}
  
  var thinkTagRe = regexp.MustCompile(`(?s)<think>.*?</think>`)

// StripThinkTags removes <think>...</think> blocks from LLM output
// produced by reasoning models (e.g. DeepSeek-R1, QwQ).
func StripThinkTags(s string) string {
	return strings.TrimSpace(thinkTagRe.ReplaceAllString(s, ""))
}

const SystemPrompt = `You are a website categorization engine. Given information about a website, respond with ONLY a single category name from the list below — nothing else.

Categories with descriptions:
- News/Media: News outlets, journalism platforms, blogs, magazines, podcasts, and media distribution sites
- Social Media: Social networking platforms, community forums, messaging apps, content sharing networks, and user-generated content sites
- E-Commerce: Online retail stores, marketplaces, shopping platforms, product catalogs, and payment processing sites
- Technology: Software companies, tech news, developer tools, SaaS platforms, cloud services, programming resources, and IT infrastructure
- Finance/Banking: Banks, financial institutions, investment platforms, cryptocurrency exchanges, and trading platforms
- Entertainment: Streaming services, movie/music databases, video content platforms, gaming sites, and entertainment news
- Education: Schools, universities, online learning platforms, e-learning courses, libraries, and academic resources
- Government: Government agencies, public administration sites, official state/federal/local resources, and public services
- Healthcare: Hospitals, clinics, medical professionals, telemedicine, pharmaceutical information, and wellness resources
- Security: Cybersecurity companies, security tools, vulnerability databases, threat intelligence, and security research
- hacking / phising: Sites used for malicious hacking, phishing campaigns, exploit distribution, malware hosting, or credential theft
- Adult Content: Adult entertainment and adult-oriented services
- Logistics: Shipping companies, delivery services, supply chain management, warehousing, and package tracking
- Energy: Power companies, energy utilities, renewable energy providers, and energy infrastructure
- Other: Websites that don't fit into any category or are unclear

Rules:
- Respond with exactly one category name as written above.
- Do not add punctuation, explanation, or extra words.
- If uncertain, use "Other".`

// IsValidCategory checks if a category is in the allowed list
func IsValidCategory(category string) bool {
	_, exists := CategoryDescriptions[category]
	return exists
}

// GetCategoryDescription returns the description for a given category
func GetCategoryDescription(category string) string {
	if desc, exists := CategoryDescriptions[category]; exists {
		return desc
	}
	return CategoryDescriptions["Other"]
}

// ValidateCategory validates and sanitizes a category response from the LLM.
// If the category is valid, returns it as-is.
// If invalid, returns "Other" as the safe default.
func ValidateCategory(category string) string {
	category = strings.TrimSpace(category)
	if IsValidCategory(category) {
		return category
	}
	return "Other"
}
