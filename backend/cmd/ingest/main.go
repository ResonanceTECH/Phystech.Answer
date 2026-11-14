// Command ingest bulk-loads the MIPT normative corpus into Postgres and the
// ML index. It reads ml/configs/corpus.json (manifest) plus ml/data/parsed/*.txt
// (extracted/OCR'd text) and creates documents with versions/sections/chunks.
//
// Usage:
//
//	go run ./cmd/ingest -manifest ../ml/configs/corpus.json -parsed ../ml/data/parsed -ml http://localhost:8000
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"phystech-answer/backend/internal/config"
	"phystech-answer/backend/internal/db"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

type manifestDoc struct {
	File           string   `json:"file"`
	Title          string   `json:"title"`
	DocumentNumber string   `json:"documentNumber"`
	DocumentType   string   `json:"documentType"`
	DocumentDate   string   `json:"documentDate"`
	Category       string   `json:"category"`
	LegalStatus    string   `json:"legalStatus"`
	Tags           []string `json:"tags"`
}

type manifest struct {
	Documents []manifestDoc `json:"documents"`
}

func main() {
	var (
		manifestPath = flag.String("manifest", "../ml/configs/corpus.json", "corpus manifest JSON")
		parsedDir    = flag.String("parsed", "../ml/data/parsed", "dir with parsed txt files")
		mlURL        = flag.String("ml", "", "ML service URL to index into (empty = skip)")
		dryRun       = flag.Bool("dry-run", false, "validate only")
	)
	flag.Parse()

	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		log.Fatalf("manifest: %v", err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatalf("manifest json: %v", err)
	}

	cfg := config.Load()
	ctx := context.Background()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL, 4)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	st := store.NewPostgres(pool)

	var docsForML []model.Document
	created := 0
	for _, md := range m.Documents {
		txtPath := filepath.Join(*parsedDir, md.File)
		content, err := os.ReadFile(txtPath)
		if err != nil {
			log.Printf("skip %s: %v", md.File, err)
			continue
		}
		doc := model.Document{
			Title:          md.Title,
			Content:        string(content),
			Tags:           md.Tags,
			DocumentNumber: md.DocumentNumber,
			DocumentType:   md.DocumentType,
			Category:       md.Category,
			AccessLevel:    "internal",
			LegalStatus:    md.LegalStatus,
		}
		if *dryRun {
			log.Printf("[dry] %s (%s) — %d chars", md.Title, md.DocumentNumber, len(content))
			continue
		}
		createdDoc, err := st.CreateDocument(ctx, doc)
		if err != nil {
			log.Printf("create %s: %v", md.Title, err)
			continue
		}
		created++
		docsForML = append(docsForML, model.Document{
			ID: createdDoc.ID, Title: createdDoc.Title, Content: string(content),
			Tags: md.Tags, DocumentNumber: md.DocumentNumber,
			DocumentType: md.DocumentType, Category: md.Category,
			LegalStatus: md.LegalStatus,
		})
		log.Printf("created %s → %s", md.Title, createdDoc.ID)
	}
	log.Printf("ingest done: %d documents", created)

	if *mlURL != "" && len(docsForML) > 0 {
		if err := pushML(ctx, *mlURL, docsForML); err != nil {
			log.Fatalf("ml push: %v", err)
		}
		log.Printf("ml index updated: %d docs", len(docsForML))
	}
}

func pushML(ctx context.Context, base string, docs []model.Document) error {
	payload := map[string]any{"documents": docs}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/sync", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// Indexing + embedding 20 docs can take a while on first run.
	hc := &http.Client{Timeout: 10 * time.Minute}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ml sync %s: %s", resp.Status, string(b))
	}
	return nil
}
