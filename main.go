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

	// 3. Configure handlers
	h := &api.Handler{SB: client}

	// 4. API Routes
	http.HandleFunc("/api/auth", h.Auth)
	http.HandleFunc("/api/tasks", h.TasksHandler)
	http.HandleFunc("/api/join-family", h.JoinFamilyHandler)
	http.HandleFunc("/api/leave-family", h.LeaveFamilyHandler)

	// 5. Serve the public directory (your HTML/CSS/JS)
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)


	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error while starting server:", err)
	}
}	
