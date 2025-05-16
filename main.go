package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Create a new ServeMux
	mux := http.NewServeMux()

	// Readiness endpoint at /healthz
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files from the "assets" directory at /assets/
	assetsFS := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", assetsFS))

	// Serve files from the current directory at /app/
	appFS := http.FileServer(http.Dir("."))
	mux.Handle("/app/", http.StripPrefix("/app", appFS))

	// Start the server on port 8888
	fmt.Println("Starting server on :8888")
	err := http.ListenAndServe(":8888", mux)
	if err != nil {
		panic(err)
	}
}
