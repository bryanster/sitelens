package main

import "os"

type Config struct {
	LLMProvider string // "ollama" or "lmstudio"
	LLMURL      string
	LLMModel    string
	DBPath      string
	Port        string
}

func LoadConfig() Config {
	return Config{
		LLMProvider: getEnv("LLM_PROVIDER", "lmstudio"),
		LLMURL:      getEnv("LLM_URL", "http://localhost:1234"),
		LLMModel:    getEnv("LLM_MODEL", "qwen/qwen3.5-9b"),
		DBPath:      getEnv("DB_PATH", "./sitelens.db"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
