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


	// 4. Configure handlers
	h := &api.Handler{SB: client}

	http.HandleFunc("/api/auth", h.Auth)
	http.HandleFunc("/api/tasks", h.TasksHandler)
	http.HandleFunc("/api/join-family", h.JoinFamilyHandler)
	http.HandleFunc("/api/leave-family", h.LeaveFamilyHandler)

	// Serve the public directory (your HTML/CSS/JS)
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)
}
