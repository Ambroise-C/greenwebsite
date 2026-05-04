package main

import (
	"log"
	"mon-projet/api"
	"mon-projet/internal"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, using system variables")
	}

	// 2. Initialize Supabase client
	client := internal.InitSupabase()

	// 3. CONNECTION TEST (Before starting the server)
	users, err := client.GetUsers()
	if err != nil {
		log.Printf("❌ ERROR: %v", err)
	} else {
		log.Printf("✅ Users: %v", users)
	}

	// 4. Configure handlers
	h := &api.Handler{SB: client}

	http.HandleFunc("/api/auth", h.Auth)
	http.HandleFunc("/api/tasks", h.TasksHandler)
	http.HandleFunc("/api/join-family", h.JoinFamilyHandler)
	http.HandleFunc("/api/leave-family", h.LeaveFamilyHandler)

	// Serve the public directory (your HTML/CSS/JS)
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// API Routes

	// 5. START SERVER (This line must be the last one)
	log.Println("🚀 Server ready on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error while starting server:", err)
	}
}
