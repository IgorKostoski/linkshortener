package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "linkshortener_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)
	linksCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "linkshortener_links_created_total",
			Help: "Total number of links created",
		},
	)
	linksRedirectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "linkshortener_links_redirected_total",
			Help: "Total number of links redirected",
		},
		[]string{"short_code"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "linkshortener_http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

var db *sql.DB

type Config struct {
	AppPort      string
	ShortURLBase string
	PostgresHost string
	PostgresPort string
	PostgresUser string
	PostgresPass string
	PostgresDB   string
}

func LoadConfig() Config {
	return Config{
		AppPort:      getEnv("APP_PORT", "8080"),
		ShortURLBase: getEnv("SHORT_URL_BASE", "http://localhost:8080"),
		PostgresHost: getEnv("POSTGRES_HOST", "db"),
		PostgresPort: getEnv("POSTGRES_PORT", "5432"),
		PostgresUser: getEnv("POSTGRES_USER", "usr"),
		PostgresPass: getEnv("POSTGRES_PASSWORD", "pwd"),
		PostgresDB:   getEnv("POSTGRES_DB", "linkshortener_db"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Environment variable %s not set, using fallback: %s", key, fallback)
	return fallback
}

func initDB(cfg Config) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB)

	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Failed to open DB (attempt %d): %v", i+1, err)
			time.Sleep(3 * time.Second)
			continue
		}
		if err = db.Ping(); err == nil {
			log.Println("Connected to DB successfully.")
			break
		}
		log.Printf("Ping failed (attempt %d): %v", i+1, err)
		db.Close()
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS links (
		id SERIAL PRIMARY KEY,
		short_code VARCHAR(10) UNIQUE NOT NULL,
		long_url TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("Failed to create links table: %v", err)
	}
	log.Println("Links table initialized.")
}

func generateShortCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func shortenHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		statusCode := http.StatusOK
		defer func() {
			httpRequestsTotal.WithLabelValues(r.Method, "/shorten", fmt.Sprintf("%d", statusCode)).Inc()
			httpRequestDuration.WithLabelValues(r.Method, "/shorten").Observe(time.Since(start).Seconds())
		}()

		if r.Method != http.MethodPost {
			statusCode = http.StatusMethodNotAllowed
			http.Error(w, "Invalid request method", statusCode)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			statusCode = http.StatusInternalServerError
			http.Error(w, "Failed to read request", statusCode)
			return
		}
		defer r.Body.Close()

		longURL := string(body)
		if longURL == "" {
			statusCode = http.StatusBadRequest
			http.Error(w, "URL cannot be empty", statusCode)
			return
		}

		var shortCode string
		for i := 0; i < 5; i++ {
			candidate, err := generateShortCode(6)
			if err != nil {
				statusCode = http.StatusInternalServerError
				http.Error(w, "Failed to generate short code", statusCode)
				return
			}

			var exists bool
			err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1)", candidate).Scan(&exists)
			if err != nil {
				log.Printf("DB error checking short code: %v", err)
				continue
			}
			if !exists {
				shortCode = candidate
				break
			}
		}

		if shortCode == "" {
			statusCode = http.StatusInternalServerError
			http.Error(w, "Could not generate a unique short code", statusCode)
			return
		}

		_, err = db.Exec("INSERT INTO links (short_code, long_url) VALUES ($1, $2)", shortCode, longURL)
		if err != nil {
			statusCode = http.StatusInternalServerError
			log.Printf("DB insert error: %v", err)
			http.Error(w, "Failed to store link", statusCode)
			return
		}

		linksCreatedTotal.Inc()
		shortURL := fmt.Sprintf("%s/%s", cfg.ShortURLBase, shortCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"short_url": shortURL})
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := http.StatusFound
	path := r.URL.Path
	shortCode := path[1:]

	defer func() {
		httpRequestsTotal.WithLabelValues(r.Method, path, fmt.Sprintf("%d", statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	}()

	if r.Method != http.MethodGet {
		statusCode = http.StatusMethodNotAllowed
		http.Error(w, "Invalid request method", statusCode)
		return
	}

	if shortCode == "" || shortCode == "favicon.ico" || shortCode == "metrics" {
		statusCode = http.StatusNotFound
		http.NotFound(w, r)
		return
	}

	var longURL string
	err := db.QueryRow("SELECT long_url FROM links WHERE short_code = $1", shortCode).Scan(&longURL)
	if err != nil {
		if err == sql.ErrNoRows {
			statusCode = http.StatusNotFound
			http.NotFound(w, r)
		} else {
			statusCode = http.StatusInternalServerError
			log.Printf("DB error on redirect: %v", err)
			http.Error(w, "Server error", statusCode)
		}
		return
	}

	linksRedirectedTotal.WithLabelValues(shortCode).Inc()
	http.Redirect(w, r, longURL, http.StatusFound)
}

func UnusedFunctionPrTest(a, b, c int) int {
	if (a > 10 && b < 5) || (c == 0 && a == 0) || (b == 1 && c == 1) {
		return a + b + c
	}

	if b > 100 {
		return a * 10
	}
	return 0
}

func main() {
	cfg := LoadConfig()
	initDB(cfg)
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", shortenHandler(cfg))
	mux.HandleFunc("/", redirectHandler)
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("Server starting on port %s", cfg.AppPort)
	log.Printf("Metrics at http://localhost:%s/metrics", cfg.AppPort)

	if err := http.ListenAndServe(":"+cfg.AppPort, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

}
