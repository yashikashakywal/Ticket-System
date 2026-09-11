package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"ticket-system/internal/handlers"
	"ticket-system/internal/store"
)

// webFiles embeds the frontend (web/index.html and any future static
// assets) directly into the compiled binary, so the same Docker image and
// deployment that serves the API also serves the UI at "/" — no separate
// frontend build or hosting step required.
//
//go:embed web
var webFiles embed.FS

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-change-me")

	if jwtSecret == "dev-secret-change-me" {
		log.Println("WARNING: using default JWT_SECRET; set the JWT_SECRET env var in production")
	}

	s := store.New()
	authHandler := handlers.NewAuthHandler(s, jwtSecret, 24*time.Hour)
	ticketHandler := handlers.NewTicketHandler(s)
	requireAuth := handlers.RequireAuth(jwtSecret)

	mux := http.NewServeMux()

	// Public routes.
	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// Protected routes (require Authorization: Bearer <token>).
	mux.Handle("POST /tickets", requireAuth(http.HandlerFunc(ticketHandler.Create)))
	mux.Handle("GET /tickets", requireAuth(http.HandlerFunc(ticketHandler.List)))
	mux.Handle("GET /tickets/{id}", requireAuth(http.HandlerFunc(ticketHandler.GetByID)))
	mux.Handle("PATCH /tickets/{id}/status", requireAuth(http.HandlerFunc(ticketHandler.UpdateStatus)))

	// Frontend: serves web/index.html (and any future static assets under
	// web/) at "/". This is a fallback pattern in Go's ServeMux, so it never
	// shadows the more specific API routes registered above.
	webRoot, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatalf("failed to load embedded web assets: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(webRoot)))

	addr := ":" + port
	log.Printf("ticket-system listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
