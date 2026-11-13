package main

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const codeAlphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

//go:embed web/*
var webFiles embed.FS

type Link struct {
	ShortCode  string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt  time.Time `json:"created_at"`
	VisitCount int64     `json:"visit_count"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			short_code TEXT PRIMARY KEY,
			original_url TEXT NOT NULL,
			created_at TEXT NOT NULL,
			visit_count INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create urls table: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(originalURL string) (Link, error) {
	for attempt := 0; attempt < 5; attempt++ {
		code, err := generateCode(7)
		if err != nil {
			return Link{}, err
		}
		createdAt := time.Now().UTC()
		_, err = s.db.Exec(
			"INSERT INTO urls (short_code, original_url, created_at) VALUES (?, ?, ?)",
			code, originalURL, createdAt.Format(time.RFC3339Nano),
		)
		if err == nil {
			return Link{ShortCode: code, OriginalURL: originalURL, CreatedAt: createdAt}, nil
		}
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Link{}, fmt.Errorf("save URL: %w", err)
		}
	}
	return Link{}, errors.New("could not generate a unique short code")
}

func (s *Store) Resolve(code string) (Link, error) {
	var link Link
	var createdAt string
	err := s.db.QueryRow(
		"SELECT short_code, original_url, created_at, visit_count FROM urls WHERE short_code = ?",
		code,
	).Scan(&link.ShortCode, &link.OriginalURL, &createdAt, &link.VisitCount)
	if err != nil {
		return Link{}, err
	}
	link.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Link{}, fmt.Errorf("parse created timestamp: %w", err)
	}
	return link, nil
}

func (s *Store) RecordVisit(code string) error {
	_, err := s.db.Exec("UPDATE urls SET visit_count = visit_count + 1 WHERE short_code = ?", code)
	return err
}

func generateCode(length int) (string, error) {
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate short code: %w", err)
	}
	code := make([]byte, length)
	for i, value := range random {
		code[i] = codeAlphabet[int(value)%len(codeAlphabet)]
	}
	return string(code), nil
}

func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("enter a valid URL including https://")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("only http and https URLs are supported")
	}
	return nil
}

type server struct {
	store *Store
}

func (s *server) routes() (http.Handler, error) {
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		return nil, fmt.Errorf("load web files: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(web)))
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /api/urls", s.createURL)
	mux.HandleFunc("GET /api/urls/{code}/stats", s.stats)
	mux.HandleFunc("GET /{code}", s.redirect)
	return logging(mux), nil
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) createURL(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	request.URL = strings.TrimSpace(request.URL)
	if err := validateURL(request.URL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	link, err := s.store.Create(request.URL)
	if err != nil {
		log.Printf("create URL: %v", err)
		writeError(w, http.StatusInternalServerError, "could not create short link")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"code":      link.ShortCode,
		"short_url": absoluteURL(r, link.ShortCode),
	})
}

func (s *server) stats(w http.ResponseWriter, r *http.Request) {
	link, err := s.store.Resolve(r.PathValue("code"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "short link not found")
		return
	}
	if err != nil {
		log.Printf("get stats: %v", err)
		writeError(w, http.StatusInternalServerError, "could not load link statistics")
		return
	}
	writeJSON(w, http.StatusOK, link)
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" || strings.Contains(code, ".") {
		http.NotFound(w, r)
		return
	}
	link, err := s.store.Resolve(code)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("resolve URL: %v", err)
		http.Error(w, "could not resolve short link", http.StatusInternalServerError)
		return
	}
	if err := s.store.RecordVisit(code); err != nil {
		log.Printf("record visit for %s: %v", code, err)
	}
	http.Redirect(w, r, link.OriginalURL, http.StatusFound)
}

func absoluteURL(r *http.Request, code string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/" + code
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func main() {
	dataDir := os.Getenv("QUICKLINK_DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("create data directory: %v", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "quicklink.db"))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	store, err := NewStore(db)
	if err != nil {
		log.Fatal(err)
	}
	srv := &server{store: store}
	handler, err := srv.routes()
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("QUICKLINK_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("QuickLink listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
