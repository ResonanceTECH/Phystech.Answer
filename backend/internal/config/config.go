package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type Config struct {
	Addr         string
	CORSOrigin   string
	MLServiceURL string
	SeedDemoData bool
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Addr:         ":" + getEnv("PORT", "8080"),
		CORSOrigin:   getEnv("CORS_ORIGIN", "*"),
		MLServiceURL: strings.TrimRight(getEnv("ML_SERVICE_URL", "http://localhost:8000"), "/"),
		SeedDemoData: getEnv("SEED_DEMO_DATA", "true") != "false",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
