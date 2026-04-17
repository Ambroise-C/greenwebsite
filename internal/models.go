package internal

// User correspond à ton db.users
type User struct {
	Username     string `json:"username"` // Clé primaire
	Password     string `json:"password"`
	Code         string `json:"code"`
	FamilyID     string `json:"family_id"`
}

// Task correspond à tes objets tâches
type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Completed   bool      `json:"completed"`
	CompletedBy string    `json:"completed_by,omitempty"`
	UserID      string    `json:"user_id,omitempty"`   // Pour les tâches privées
	FamilyID    string    `json:"family_id,omitempty"` // Pour les tâches famille
	Scope       string    `json:"scope"`               // "private" ou "family"
}

// Family correspond à ton db.families
type Family struct {
	Owner   string   `json:"owner"`
	Members []string `json:"members"`
}