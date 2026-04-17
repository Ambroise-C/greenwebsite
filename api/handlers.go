package api

import (
	"encoding/json"
	"log"
	"mon-projet/internal"
	"net/http"
	"strconv" // Nécessaire pour convertir les ID int64 en string

	"github.com/nedpals/supabase-go"
)

type Handler struct {
	SB *supabase.Client
}

// Auth gère la connexion et la création automatique de compte
// Auth gère la connexion (Login) et l'inscription automatique (Register)
func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
	
	// 1. On décode ce que le JS nous envoie
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	// 2. On cherche si ce pseudo existe déjà dans la table "users"
	var users []internal.User
	err := h.SB.DB.From("users").Select("*").Eq("username", creds.User).Execute(&users)

	var user internal.User

	// CAS A : L'utilisateur n'existe PAS (Inscription)
	if err != nil || len(users) == 0 {
		log.Printf("Nouvel utilisateur détecté : %s. Création du compte...", creds.User)
		
		user = internal.User{
			Username: creds.User,
			Password: creds.Pass,
			Code:     "START123", // Code par défaut
			FamilyID: 0,          // Pas de famille au début
		}

		// On insère le nouveau profil
		var insertedUsers []internal.User
		errInsert := h.SB.DB.From("users").Insert(user).Execute(&insertedUsers)
		
		if errInsert != nil {
			log.Printf("❌ Erreur lors de la création : %v", errInsert)
			http.Error(w, "Impossible de créer le compte", http.StatusInternalServerError)
			return
		}
		
		if len(insertedUsers) > 0 {
			user = insertedUsers[0]
		}
		log.Println("✅ Compte créé avec succès")

	} else {
		// CAS B : L'utilisateur existe déjà (Connexion)
		user = users[0]

		// VERIFICATION DU MOT DE PASSE
		if user.Password != creds.Pass {
			log.Printf("⚠️ Tentative de connexion échouée pour %s : mauvais mot de passe", creds.User)
			// On renvoie une erreur 401 (Unauthorized)
			http.Error(w, "Pseudo déjà utilisé ou mot de passe incorrect", http.StatusUnauthorized)
			return
		}
		log.Printf("✅ Connexion réussie pour %s", creds.User)
	}

	// 3. On renvoie l'utilisateur (avec son user_ID tout neuf) au JS
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// TasksHandler gère l'ajout (POST) et la récupération (GET) des tâches
func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	// 1. On récupère le pseudo depuis la requête pour identifier qui parle
	username := r.URL.Query().Get("user")

	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", username).Execute(&users)

	if len(users) == 0 {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}
	dbUser := users[0]

	// --- CAS POST : AJOUTER UNE TÂCHE ---
	if r.Method == http.MethodPost {
		var newTask internal.Task
		if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
			http.Error(w, "Erreur format JSON", http.StatusBadRequest)
			return
		}

		// On injecte les IDs numériques officiels de la DB
		newTask.UserID = dbUser.UserID
		newTask.FamilyID = dbUser.FamilyID
		newTask.Completed = false

		err := h.SB.DB.From("tasks").Insert(newTask).Execute(nil)
		if err != nil {
			log.Printf("❌ Erreur Supabase Insert: %v", err)
			http.Error(w, "Erreur insertion tâche", 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	// --- CAS GET : LIRE LES TÂCHES ---
	var privateTasks []internal.Task
	var familyTasks []internal.Task

	// Conversion des int64 en string pour les filtres .Eq() de Supabase
	userIDStr := strconv.FormatInt(dbUser.UserID, 10)

	// Récupération des tâches privées
	h.SB.DB.From("tasks").Select("*").Eq("user_ID", userIDStr).Eq("scope", "private").Execute(&privateTasks)

	// Récupération des tâches famille (si l'ID famille est différent de 0)
	if dbUser.FamilyID != 0 {
		familyIDStr := strconv.FormatInt(dbUser.FamilyID, 10)
		h.SB.DB.From("tasks").Select("*").Eq("family_ID", familyIDStr).Eq("scope", "family").Execute(&familyTasks)
	}

	// On renvoie tout au JS (incluant l'user pour mettre à jour le frontend)
	response := map[string]interface{}{
		"user":    dbUser,
		"private": privateTasks,
		"family":  familyTasks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}