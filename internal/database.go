package internal

import (
	"os"

	"github.com/nedpals/supabase-go"
)

// InitSupabase creates an instance of the client with the environment variables
func InitSupabase() *supabase.Client {
	sbUrl := os.Getenv("SUPABASE_URL")
	sbKey := os.Getenv("SUPABASE_KEY")
	return supabase.CreateClient(sbUrl, sbKey)
}
