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
	MLServiceURL string
	MLTimeoutSec int
	SeedDemoData bool
	DocsDir      string

	DatabaseURL string
	DBMaxConns  int32

	StorageDriver string
	StoragePath   string

	WorkerConcurrency int
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Addr:              ":" + getEnv("PORT", getEnv("HTTP_PORT", "8080")),
		CORSOrigin:        getEnv("CORS_ORIGIN", "*"),
		MLServiceURL:      strings.TrimRight(getEnv("ML_SERVICE_URL", "http://localhost:8000"), "/"),
		MLTimeoutSec:      getEnvInt("ML_TIMEOUT_SECONDS", 120),
		SeedDemoData:      getEnv("SEED_DEMO_DATA", "false") == "true",
		DocsDir:           getEnv("DOCS_DIR", ""),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		DBMaxConns:        int32(getEnvInt("DB_MAX_CONNS", 10)),
		StorageDriver:     getEnv("STORAGE_DRIVER", "local"),
		StoragePath:       getEnv("STORAGE_PATH", "./storage"),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 2),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
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
