package internal

import (
    "os"
    "github.com/nedpals/supabase-go"
)

// InitSupabase crée une instance du client avec les variables d'env
func InitSupabase() *supabase.Client {
    sbUrl := os.Getenv("SUPABASE_URL")
    sbKey := os.Getenv("SUPABASE_KEY")
    return supabase.CreateClient(sbUrl, sbKey)
}