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
	LLMReasoning string
	EmbedBaseURL string
	EmbedAPIKey  string
	EmbedModel   string
	IndexPath    string
	TopK         int

	RetrievalMode string  // bm25|vector|hybrid
	Reranker      string  // off | llm | decision
	DecisionURL   string  // /v1/systemone server (H2O-Lightning on llama-server)
	Amendments    bool    // pull in amending acts
	Decompose     bool    // split composite questions (LLM)
	MinVector     float64 // evidence gate (cosine of best fragment)
	MinBM25       float64 // evidence gate (BM25 of best fragment)
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Addr:         ":" + getEnv("PORT", "8000"),
		CORSOrigin:   getEnv("CORS_ORIGIN", "*"),
		LLMBaseURL:   envURL("LLM_BASE_URL", "http://localhost:11434/v1"),
		LLMAPIKey:    getEnv("LLM_API_KEY", ""),
		LLMModel:     getEnv("LLM_MODEL", "qwen3.5:4b"),
		LLMReasoning: os.Getenv("LLM_REASONING_EFFORT"),
		EmbedBaseURL: envURL("EMBED_BASE_URL", "http://localhost:11434/v1"),
		EmbedAPIKey:  getEnv("EMBED_API_KEY", ""),
		EmbedModel:   getEnv("EMBED_MODEL", "bge-m3"),
		IndexPath:    getEnv("INDEX_PATH", "index.json"),
		TopK:         getEnvInt("RAG_TOP_K", 6),

		RetrievalMode: getEnv("RAG_MODE", "hybrid"),
		Reranker:      rerankerFromEnv(getEnv("RAG_RERANK", "llm")),
		DecisionURL:   envURL("DECISION_BASE_URL", "off"),
		Amendments:    getEnv("RAG_AMENDMENTS", "true") == "true",
		Decompose:     getEnv("RAG_DECOMPOSE", "true") == "true",
		MinVector:     getEnvFloat("RAG_MIN_VECTOR", 0.47),
		MinBM25:       getEnvFloat("RAG_MIN_BM25", 10.0),
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

// rerankerFromEnv accepts off|llm|decision (and legacy true/false).
func rerankerFromEnv(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "off", "false", "none", "0":
		return ""
	case "decision", "h2o":
		return "decision"
	default:
		return "llm"
	}
}
