package api

import (
	"encoding/json"
	"log"
	"mon-projet/internal"
	"net/http"

	"github.com/nedpals/supabase-go"
)

type Handler struct {
	SB *supabase.Client
}

// 1. LOGIN / REGISTER
func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", creds.User).Execute(&users)

	var user internal.User
	if len(users) == 0 {
		// Création si inexistant
		user = internal.User{
			Username: creds.User,
			Password: creds.Pass,
			Code:     "ABC456", // À générer dynamiquement plus tard
			FamilyID: creds.User,
		}
		err := h.SB.DB.From("users").Insert(user).Execute(nil)
		if err != nil {
			log.Printf("Erreur insertion user: %v", err)
		}
	} else {
		user = users[0]
		if user.Password != creds.Pass {
			http.Error(w, "MDP Incorrect", http.StatusUnauthorized)
			return
		}
	}
	json.NewEncoder(w).Encode(user)
}

// 2. RECUPERER ET AJOUTER DES TÂCHES
func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	// CAS : AJOUTER UNE TÂCHE (POST)
	if r.Method == http.MethodPost {
		var newTask internal.Task
		if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
			http.Error(w, "Erreur JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Tentative d'ajout de tâche: %+v", newTask)

		var result []internal.Task
		err := h.SB.DB.From("tasks").Insert(newTask).Execute(&result)
		if err != nil {
			log.Printf("❌ Erreur Supabase: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("✅ Tâche ajoutée avec succès")
		w.WriteHeader(http.StatusCreated)
		return
	}

	// CAS : LIRE LES TÂCHES (GET)
	username := r.URL.Query().Get("user")
	
	// On récupère d'abord l'utilisateur pour avoir son FamilyID à jour
	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", username).Execute(&users)
	
	if len(users) == 0 {
		http.Error(w, "User non trouvé", http.StatusNotFound)
		return
	}
	user := users[0]

	var privateTasks []internal.Task
	var familyTasks []internal.Task

	h.SB.DB.From("tasks").Select("*").Eq("user_id", username).Eq("scope", "private").Execute(&privateTasks)
	h.SB.DB.From("tasks").Select("*").Eq("family_id", user.FamilyID).Eq("scope", "family").Execute(&familyTasks)

	response := map[string]interface{}{
		"user":    user, // Crucial pour que le JS reçoive le family_id !
		"private": privateTasks,
		"family":  familyTasks,
	}
	json.NewEncoder(w).Encode(response)
}