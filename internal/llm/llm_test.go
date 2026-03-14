package llm

import (
	"testing"
)

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Valid categories
		{"News/Media", "News/Media"},
		{"Social Media", "Social Media"},
		{"E-Commerce", "E-Commerce"},
		{"Technology", "Technology"},
		{"Finance/Banking", "Finance/Banking"},
		{"Entertainment", "Entertainment"},
		{"Education", "Education"},
		{"Government", "Government"},
		{"Healthcare", "Healthcare"},
		{"Security", "Security"},
		{"hacking / phising", "hacking / phising"},
		{"Adult Content", "Adult Content"},
		{"Logistics", "Logistics"},
		{"Energy", "Energy"},
		{"Other", "Other"},

		// Invalid categories should default to "Other"
		{"InvalidCategory", "Other"},
		{"news", "Other"}, // case-sensitive
		{"News / Media", "Other"}, // extra spaces
		{"", "Other"},
		{"   ", "Other"},
		{"Technology123", "Other"},

		// Valid categories with whitespace should be trimmed
		{"  News/Media  ", "News/Media"},
		{"\tEducation\n", "Education"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidateCategory(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateCategory(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"News/Media", true},
		{"Social Media", true},
		{"InvalidCategory", false},
		{"news", false},
		{"", false},
		{"Other", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsValidCategory(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidCategory(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
