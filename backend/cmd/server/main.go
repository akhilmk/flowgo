package main

import (
	"log"
	"net/http"
	"os"

	"github.com/akhilmk/vectorgo/internal/auth"
	"github.com/akhilmk/vectorgo/internal/document"
)

func main() {
	port := getEnv("PORT", "8080")
	log.Printf("VectorGo server starting on :%s...", port)

	mux := http.NewServeMux()

	// Initialize Handlers (Config loaded internally)
	authHandler := auth.NewHandler()
	docHandler := document.NewHandler()

	// Register Routes
	authHandler.RegisterRoutes(mux)
	docHandler.RegisterRoutes(mux, authHandler.Middleware)

	// Public Health Check
	mux.HandleFunc("/api/health", handleHealth)

	// Serve Frontend
	fs := http.FileServer(http.Dir("frontend/dist"))
	mux.Handle("/", fs)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","service":"VectorGo","version":"1.0.0"}`))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
