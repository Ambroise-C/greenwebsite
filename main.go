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
    var testData []map[string]interface{}
    err := client.DB.From("users").Select("*").Limit(1).Execute(&testData)

    if err != nil {
        log.Printf("❌ ERROR: Unable to read row: %v", err)
    } else if len(testData) > 0 {
        log.Println("✅ ROW RETRIEVED:")
        // %+v displays both map keys and values
        log.Printf("%+v", testData[0])
    } else {
        log.Println("✅ CONNECTION OK: But the table is empty (no rows to display).")
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