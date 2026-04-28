package api

import (
	"encoding/json"
	"log"
	"mon-projet/internal"
	"net/http"
	"strconv"
	"math/rand"
	"time"
	"fmt"
	"github.com/nedpals/supabase-go"
)

type Handler struct {
	SB *supabase.Client
}

func generateShortCode(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func (h *Handler) getNewFamilyID() int64 {
    var allIds []struct {
        FamilyID int64 `json:"family_ID"`
    }

    err := h.SB.DB.From("families").Select("family_ID").Execute(&allIds)
    
    var maxID int64 = 0
    
    if err == nil && len(allIds) > 0 {
        for _, f := range allIds {
			log.Printf("comparing family_ID : %d with current max_ID %d", f.FamilyID, maxID)
            if f.FamilyID > maxID {
                maxID = f.FamilyID
            }
        }
    } else if err != nil {
        log.Printf("Erreur lors de la récupération des IDs : %v", err)
    }
	log.Printf("Max id founded is %d, and we will use %d as an id", maxID, maxID +1)
    return maxID + 1
}

func (h *Handler) getNewUserID() int64 {
    var allIds []struct {
        UserID int64 `json:"user_ID"`
    }

    err := h.SB.DB.From("users").Select("user_ID").Execute(&allIds)
    
    var maxID int64 = 0
    
    if err == nil && len(allIds) > 0 {
        for _, f := range allIds {
			log.Printf("comparing user_ID : %d with current max_ID %d", f.UserID, maxID)
            if f.UserID > maxID {
                maxID = f.UserID
            }
        }
    } else if err != nil {
        log.Printf("Erreur lors de la récupération des IDs : %v", err)
    }
	log.Printf("Max id founded is %d, and we will use %d as an id", maxID, maxID +1)
    return maxID + 1
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

		new_family_ID := h.getNewFamilyID()
		new_user_ID := h.getNewUserID()


		log.Printf("Création compte : %s", creds.User)
		user = internal.User{
			UserID: new_user_ID,
			Username: creds.User,
			Password: creds.Pass,
			FamilyID: new_family_ID,
		}

		var inserted []internal.User
		err := h.SB.DB.From("users").Insert(user).Execute(&inserted)
		if err != nil || len(inserted) == 0 {
			log.Printf("Erreur insertion user: %v", err)
			http.Error(w, "Erreur création", 500)
			return
		}
		user = inserted[0]

		var existingCodes []struct{ Code string }
		err = h.SB.DB.From("families").Select("code").Execute(&existingCodes)
		if err != nil {
			log.Printf("Avertissement : impossible de vérifier l'unicité via la liste complète : %v", err)
		}
		codeMap := make(map[string]bool)
		for _, f := range existingCodes {
			codeMap[f.Code] = true
		}
		familyCode := generateShortCode(7)
		for codeMap[familyCode] {
			familyCode = generateShortCode(7)
		}

		newFamily := internal.Family{
			FamilyID: new_family_ID,
			OwnerID:  new_user_ID,
			Members:  []string{user.Username},
			Code:     familyCode,
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

	// POST

	if r.Method == http.MethodPost {
		var newTask internal.Task
		if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
			http.Error(w, "JSON invalide", 400)
			return
		}
		fmt.Printf("JSON reçu et décodé : %+v\n", newTask)
		newTask.UserID = dbUser.UserID
		newTask.Completed = false

		if newTask.Scope == "private" {
			newTask.FamilyID = nil 
		} else {
			familyIDCopy := dbUser.FamilyID
			newTask.FamilyID = &familyIDCopy
		}

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
	fID := strconv.FormatInt(dbUser.FamilyID, 10)
	h.SB.DB.From("tasks").Select("*").Eq("family_ID", fID).Eq("scope", "family").Execute(&family)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":    dbUser,
		"private": private,
		"family":  family,
	})

	if r.Method == http.MethodDelete {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "ID manquant", 400)
			return
		}

		err := h.SB.DB.From("tasks").Delete().Eq("task_ID", idStr).Execute(nil)
		if err != nil {
			log.Printf("Erreur suppression DB: %v", err)
			http.Error(w, "Erreur serveur", 500)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}