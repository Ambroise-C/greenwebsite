package internal

type User struct {
    UserID   int64  `json:"user_ID" db:"user_ID"`
    Username string `json:"username" db:"username"`
    Password string `json:"password" db:"password"`
    Code     string `json:"code" db:"code"`
    FamilyID int64  `json:"family_ID" db:"family_ID"` // Changé en int64
}

type Task struct {
    TaskID      int64  `json:"task_ID,omitempty" db:"task_ID"`
    Title       string `json:"title" db:"title"`
    Completed   bool   `json:"completed" db:"completed"`
    CompletedBy string `json:"completedBy" db:"completedBy"`
    UserID      int64  `json:"user_ID" db:"user_ID"`
    FamilyID    int64  `json:"family_ID" db:"family_ID"`
    Scope       string `json:"scope" db:"scope"`
}

type Family struct {
    FamilyID int64    `json:"family_ID" db:"family_ID"`
    OwnerID  int64    `json:"owner_ID" db:"owner_ID"`
    Members  []string `json:"members" db:"members"`
}