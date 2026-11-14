package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr         string
	CORSOrigin   string
	LLMBaseURL   string
	LLMAPIKey    string
	LLMModel     string
	EmbedBaseURL string
	EmbedAPIKey  string
	EmbedModel   string
	IndexPath    string
	TopK         int
	MinScore     float64
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Addr:         ":" + getEnv("PORT", "8000"),
		CORSOrigin:   getEnv("CORS_ORIGIN", "*"),
		LLMBaseURL:   envURL("LLM_BASE_URL", "http://localhost:11434/v1"),
		LLMAPIKey:    getEnv("LLM_API_KEY", ""),
		LLMModel:     getEnv("LLM_MODEL", "qwen2.5:7b"),
		EmbedBaseURL: envURL("EMBED_BASE_URL", "http://localhost:11434/v1"),
		EmbedAPIKey:  getEnv("EMBED_API_KEY", ""),
		EmbedModel:   getEnv("EMBED_MODEL", "nomic-embed-text"),
		IndexPath:    getEnv("INDEX_PATH", "index.json"),
		TopK:         getEnvInt("RAG_TOP_K", 8),
		MinScore:     getEnvFloat("RAG_MIN_SCORE", 0.25),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envURL reads a URL env var. The value "off" disables the integration.
func envURL(key, fallback string) string {
	v := strings.TrimRight(getEnv(key, fallback), "/")
	if strings.EqualFold(v, "off") || strings.EqualFold(v, "none") {
		return ""
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// loadDotEnv is a minimal .env loader: KEY=VALUE lines, '#' comments.
// Existing environment variables take precedence.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			log.Printf("config: failed to set %s: %v", key, err)
		}
	}
}
