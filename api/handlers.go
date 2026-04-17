package api

import (
	"encoding/json"
	"log"
	"mon-projet/internal"
	"net/http"
	"strconv"

	"github.com/nedpals/supabase-go"
)

type Handler struct {
	SB *supabase.Client
}

func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", creds.User).Execute(&users)

	var user internal.User
	if len(users) == 0 {
		log.Printf("Création compte : %s", creds.User)
		user = internal.User{
			Username: creds.User,
			Password: creds.Pass,
			Code:     "ABC123", 
			FamilyID: 0,
		}

		// 1. Insertion de l'utilisateur
		var inserted []internal.User
		err := h.SB.DB.From("users").Insert(user).Execute(&inserted)
		if err != nil || len(inserted) == 0 {
			log.Printf("Erreur insertion user: %v", err)
			http.Error(w, "Erreur création", 500)
			return
		}
		user = inserted[0]

		// --- DÉBUT CRÉATION FAMILLE AUTOMATIQUE ---
		
		// 2. Préparation de la nouvelle famille (Owner = l'utilisateur actuel)
		newFamily := internal.Family{
			OwnerID: user.UserID,
			Members: []string{user.Username},
		}

		// 3. Insertion de la famille dans Supabase
		var insertedFamilies []internal.Family
		errFam := h.SB.DB.From("families").Insert(newFamily).Execute(&insertedFamilies)

		if errFam == nil && len(insertedFamilies) > 0 {
			createdFamily := insertedFamilies[0]
			user.FamilyID = createdFamily.FamilyID

			// 4. On met à jour l'utilisateur pour le lier à sa nouvelle famille
			uIDStr := strconv.FormatInt(user.UserID, 10)
			h.SB.DB.From("users").Update(map[string]interface{}{
				"family_ID": user.FamilyID,
			}).Eq("user_ID", uIDStr).Execute(nil)

			log.Printf("✅ Famille #%d créée pour %s", user.FamilyID, user.Username)
		} else {
			log.Printf("⚠️ Erreur création famille: %v", errFam)
		}

		// --- FIN CRÉATION FAMILLE ---

	} else {
		user = users[0]
		if user.Password != creds.Pass {
			http.Error(w, "MDP incorrect", 401)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	log.Printf("Requête tasks pour : [%s]", username)

	if username == "" {
		log.Printf("le bug")
		http.Error(w, "User manquant", 400)
		return
	}

	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", username).Execute(&users)

	if len(users) == 0 {
		http.Error(w, "Utilisateur non trouvé", 404)
		return
	}
	dbUser := users[0]

	if r.Method == http.MethodPost {
		var newTask internal.Task
		if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
			http.Error(w, "JSON invalide", 400)
			return
		}
		
		newTask.UserID = dbUser.UserID
		newTask.FamilyID = dbUser.FamilyID
		newTask.Completed = false

		err := h.SB.DB.From("tasks").Insert(newTask).Execute(nil)
		if err != nil {
			log.Printf("Erreur Insert: %v", err)
			http.Error(w, "Erreur insert", 500)
			return
		}
		w.WriteHeader(201)
		return
	}

	// GET
	var private []internal.Task
	var family []internal.Task

	uID := strconv.FormatInt(dbUser.UserID, 10)
	h.SB.DB.From("tasks").Select("*").Eq("user_ID", uID).Eq("scope", "private").Execute(&private)

	if dbUser.FamilyID != 0 {
		fID := strconv.FormatInt(dbUser.FamilyID, 10)
		h.SB.DB.From("tasks").Select("*").Eq("family_ID", fID).Eq("scope", "family").Execute(&family)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":    dbUser,
		"private": private,
		"family":  family,
	})
}