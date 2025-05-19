package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"os"

	"time"
)

var db *sql.DB

type Config struct {
	AppPort      string
	PostgresHost string
	PostgresPort string
	PostgresUser string
	PostgresPass string
	PostgresDB   string
}

func LoadConfig() Config {
	return Config{
		AppPort:      getEnv("APP_PORT", "8080"),
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

// Initialize the database connection
func initDB(cfg Config) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB)

	var err error
	// Retry connecting to the database a few times
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Failed to open database connection (attempt %d/5): %v\n", i+1, err)
			time.Sleep(3 * time.Second)
			continue
		}
		err = db.Ping()
		if err == nil {
			log.Println("Successfully connected to the database!")
			break
		}
		log.Printf("Failed to ping database (attempt %d/5): %v\n", i+1, err)
		if db != nil {
			db.Close()
		}
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Could not connect to the database after several attempts: %v\n", err)
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS links (
        id SERIAL PRIMARY KEY,
        short_code VARCHAR(10) UNIQUE NOT NULL,
        long_url TEXT NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create links table: %v\n", err)
	}
	log.Println("Links table checked/created successfully.")

	log.Println("Attempting to create/check links table...")
	res, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to execute create links table statement: %v", err)
	}
	rowsAffected, _ := res.RowsAffected() // For CREATE TABLE this might be 0 or driver-dependent
	log.Printf("Links table statement executed. Rows affected (if applicable): %d", rowsAffected)
	log.Println("Links table checked/created successfully.")
}

func generateShortCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request, cfg Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body) // Renamed to 'err' to avoid confusion with 'codeErr'
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	longURL := string(body)
	if longURL == "" {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	var shortCode string
	var generated bool // Renamed from 'generated'
	// Loop to generate a unique short code
	for i := 0; i < 5; i++ {
		var currentShortCode string
		var genErr error // Use a distinct error variable for generation
		currentShortCode, genErr = generateShortCode(6)
		if genErr != nil {
			log.Printf("Error generating short code: %v", genErr) // Log the error
			http.Error(w, "Failed to generate short code", http.StatusInternalServerError)
			return
		}

		// Check if shortCode already exists in DB
		var exists bool
		// Use 'err' for the database query error, distinct from 'genErr'
		queryErr := db.QueryRow("SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1)", currentShortCode).Scan(&exists)
		if queryErr != nil {
			log.Printf("Error checking if short code '%s' exists: %v", currentShortCode, queryErr)
			// Decide if you want to retry or fail here. Retrying is reasonable for transient DB issues.
			// If you continue, the loop will try to generate a new code.
			continue
		}

		if !exists {
			shortCode = currentShortCode // Assign to the outer scope shortCode
			generated = true
			break // Exit loop, unique code found
		}
		// If code exists, loop continues to generate a new one
	}

	if !generated || shortCode == "" { // Check if a code was successfully generated and assigned
		http.Error(w, "Could not generate a unique short code after several attempts, try again.", http.StatusInternalServerError)
		return
	}

	// Insert into database
	insertSQL := `INSERT INTO links (short_code, long_url) VALUES ($1, $2)`
	_, err = db.Exec(insertSQL, shortCode, longURL) // Re-use 'err' for this operation
	if err != nil {
		log.Printf("Error inserting link (short_code: %s) into database: %v", shortCode, err)
		http.Error(w, "Failed to store link", http.StatusInternalServerError)
		return
	}

	log.Printf("Shortened: %s -> %s\n", shortCode, longURL)
	fullShortURL := fmt.Sprintf("http://localhost:%s/%s", cfg.AppPort, shortCode)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, fullShortURL)
}

// Handler for redirecting a short URL
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	shortCode := r.URL.Path[1:]

	if shortCode == "" {

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Welcome to the Link Shortener API! Use POST /shorten to create a link.")
		return
	}
	if shortCode == "favicon.ico" {
		http.NotFound(w, r)
		return
	}

	var longURL string
	querySQL := `SELECT long_url FROM links WHERE short_code = $1`
	err := db.QueryRow(querySQL, shortCode).Scan(&longURL)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Short code not found: %s\n", shortCode)
			http.NotFound(w, r)
		} else {
			log.Printf("Error retrieving link from database for short_code '%s': %v\n", shortCode, err)
			http.Error(w, "Error retrieving link", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Redirecting: %s -> %s\n", shortCode, longURL)
	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	cfg := LoadConfig()
	initDB(cfg)
	if db != nil {
		defer db.Close()
	}

	// Pass config to shortenHandler
	http.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		shortenHandler(w, r, cfg)
	})
	http.HandleFunc("/", redirectHandler)

	log.Printf("Server starting on port %s\n", cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, nil); err != nil {
		log.Fatalf("Failed to start server: %s\n", err)
	}
}
