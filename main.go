package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

// Structures for JSON handling
type ChirpRequest struct {
	Body string `json:"body"`
}

type CleanedResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Helper functions for HTTP responses
func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, ErrorResponse{Error: msg})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// Helper function to clean profanity
func cleanProfanity(input string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(input, " ")

	for i, word := range words {
		wordLower := strings.ToLower(word)
		for _, profane := range profaneWords {
			if wordLower == profane {
				words[i] = "****"
				break
			}
		}
	}

	return strings.Join(words, " ")
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
}

func (cfg *apiConfig) adminResetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var chirp ChirpRequest
	err := decoder.Decode(&chirp)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}

	if len(chirp.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	// Clean profanity and respond with cleaned text
	cleanedBody := cleanProfanity(chirp.Body)
	respondWithJSON(w, http.StatusOK, CleanedResponse{CleanedBody: cleanedBody})
}

func main() {
	// Create API config
	apiCfg := &apiConfig{}

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Readiness endpoint at /api/healthz - GET only
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Validate chirp endpoint - POST only
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

	// Admin metrics endpoint - GET only, returns HTML
	mux.HandleFunc("GET /admin/metrics", apiCfg.adminMetricsHandler)

	// Admin reset endpoint - POST only
	mux.HandleFunc("POST /admin/reset", apiCfg.adminResetHandler)

	// Serve static files from the "assets" directory at /assets/
	assetsFS := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", assetsFS))

	// Serve files from the current directory at /app/
	appFS := http.FileServer(http.Dir("."))
	// Wrap the file server with the metrics middleware
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", appFS)))

	// Start the server on port 8888
	fmt.Println("Starting server on :8888")
	err := http.ListenAndServe(":8888", mux)
	if err != nil {
		panic(err)
	}
}
