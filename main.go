package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

var urlStore = &sync.Map{}

func generateShortCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
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

	// Generate a unique short code
	var finalShortCode string
	var generatedCode string
	var codeErr error
	foundUnique := false
	maxRetries := 5 // Max attempts to find a unique code

	for i := 0; i < maxRetries; i++ {
		generatedCode, codeErr = generateShortCode(6) // 6-character short code
		if codeErr != nil {
			log.Printf("Error generating short code: %v", codeErr)
			http.Error(w, "Failed to generate short code internally", http.StatusInternalServerError)
			return
		}
		// Check if the generated code already exists in the store
		if _, loaded := urlStore.Load(generatedCode); !loaded {
			finalShortCode = generatedCode // Assign to finalShortCode
			foundUnique = true
			break // Exit loop, unique code found
		}
		// If code exists, log it and loop will continue to try again
		log.Printf("Collision detected for generated code '%s', retrying (%d/%d)...", generatedCode, i+1, maxRetries)
	}

	if !foundUnique {
		log.Println("Could not generate a unique short code after multiple retries.")
		http.Error(w, "Could not generate a unique short code, please try again.", http.StatusInternalServerError)
		return
	}

	// Store the successfully generated unique short code
	urlStore.Store(finalShortCode, longURL)
	log.Printf("Shortened: %s -> %s\n", finalShortCode, longURL)

	// Respond with the short URL
	fullShortURL := fmt.Sprintf("http://localhost:8080/%s", finalShortCode)
	w.Header().Set("Content-Type", "text/plain") // Important for curl to display it as plain text
	fmt.Fprintf(w, fullShortURL)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	shortCode := r.URL.Path[1:]
	if shortCode == "" {
		// If path is just "/", treat as not found or serve a homepage
		// For now, we assume any path other than /shorten is a shortCode attempt
		// If you want a specific homepage for "/", handle it before this.
		if r.URL.Path == "/" {
			http.Error(w, "Welcome! Use /shorten to create a link.", http.StatusOK)
			return
		}
		http.NotFound(w, r)
		return
	}

	longURLInterface, ok := urlStore.Load(shortCode)
	if !ok {
		http.NotFound(w, r)
		return
	}

	longURL := longURLInterface.(string)
	log.Printf("Redirecting: %s -> %s\n", shortCode, longURL)
	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/", redirectHandler)

	port := "8080"
	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %s\n", err)
	}
}
