package api

import (
	"encoding/json"
	"log"
	"math/rand"
	"mon-projet/internal"
	"net/http"
	"strconv"
	"time"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	SB *internal.SupabaseClient
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
	results, err := h.SB.SelectFrom("families", "family_ID", map[string]interface{}{})

	var maxID int64 = 0

	if err == nil && len(results) > 0 {
		for _, f := range results {
			if fID, ok := f["family_ID"].(float64); ok {
				familyID := int64(fID)
				log.Printf("comparing family_ID: %d with current max_ID %d", familyID, maxID)
				if familyID > maxID {
					maxID = familyID
				}
			}
		}
	} else if err != nil {
		log.Printf("Error retrieving IDs: %v", err)
	}
	log.Printf("Max id found is %d, and we will use %d as an id", maxID, maxID+1)
	return maxID + 1
}

func (h *Handler) getNewUserID() int64 {
	results, err := h.SB.SelectFrom("users", "user_ID", map[string]interface{}{})

	var maxID int64 = 0

	if err == nil && len(results) > 0 {
		for _, f := range results {
			if uID, ok := f["user_ID"].(float64); ok {
				userID := int64(uID)
				log.Printf("comparing user_ID: %d with current max_ID %d", userID, maxID)
				if userID > maxID {
					maxID = userID
				}
			}
		}
	} else if err != nil {
		log.Printf("Error retrieving IDs: %v", err)
	}
	log.Printf("Max id found is %d, and we will use %d as an id", maxID, maxID+1)
	return maxID + 1
}

func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
    var creds struct {
        User string `json:"user"`
        Pass string `json:"pass"`
    }
    if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    if len(creds.User) < 1 || len(creds.Pass) < 1 {
        http.Error(w, "Username and password required", http.StatusBadRequest)
        return
    }

    users, _ := h.SB.SelectFrom("users", "user_ID,username,password,family_ID", map[string]interface{}{"username": creds.User})

    var user internal.User
    if len(users) == 0 {
        new_family_ID := h.getNewFamilyID()
        new_user_ID := h.getNewUserID()

        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Pass), bcrypt.DefaultCost)
        if err != nil {
            http.Error(w, "Server error during hashing", 500)
            return
        }

        user = internal.User{
            UserID:   new_user_ID,
            Username: creds.User,
            Password: string(hashedPassword),
            FamilyID: new_family_ID,
        }

        _, err = h.SB.InsertInto("users", user)
        if err != nil {
            http.Error(w, "Creation error", 500)
            return
        }

        familyCode := generateShortCode(7)
        newFamily := internal.Family{
            FamilyID: new_family_ID,
            OwnerID:  new_user_ID,
            Members:  []string{user.Username},
            Code:     familyCode,
        }
        h.SB.InsertInto("families", newFamily)
        log.Printf("✅ Compte et Famille créés pour %s", user.Username)

    } else {
        userMap := users[0]
        if uID, ok := userMap["user_ID"].(float64); ok { user.UserID = int64(uID) }
        if uName, ok := userMap["username"].(string); ok { user.Username = uName }
        if uPass, ok := userMap["password"].(string); ok { user.Password = uPass }
        if fID, ok := userMap["family_ID"].(float64); ok { user.FamilyID = int64(fID) }

        err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Pass))
        if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        }
    }

    response := map[string]interface{}{
        "user_ID":   user.UserID,
        "username":  user.Username,
        "family_ID": user.FamilyID,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "Missing user", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	if password, ok := updates["password"].(string); ok && password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Update error", http.StatusInternalServerError)
			return
		}
		updates["password"] = string(hashedPassword)
	}

	if usernameValue, ok := updates["username"].(string); ok && usernameValue == "" {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}

	if err := h.SB.UpdateTable("users", updates, map[string]interface{}{"username": username}); err != nil {
		http.Error(w, "Update error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
}

func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "Missing user", 400)
		return
	}

	users, _ := h.SB.SelectFrom("users", "user_ID,username,password,family_ID", map[string]interface{}{"username": username})
	if len(users) == 0 {
		http.Error(w, "User not found", 404)
		return
	}

	userMap := users[0]
	var dbUser internal.User
	if userID, ok := userMap["user_ID"].(float64); ok {
		dbUser.UserID = int64(userID)
	}
	if userName, ok := userMap["username"].(string); ok {
		dbUser.Username = userName
	}
	if familyID, ok := userMap["family_ID"].(float64); ok {
		dbUser.FamilyID = int64(familyID)
	}

	switch r.Method {
	case http.MethodGet:
		private, _ := h.SB.SelectFrom("tasks", "*", map[string]interface{}{"user_ID": dbUser.UserID, "scope": "private"})

		family, _ := h.SB.SelectFrom("tasks", "*", map[string]interface{}{"family_ID": dbUser.FamilyID, "scope": "family"})

		families, _ := h.SB.SelectFrom("families", "*", map[string]interface{}{"family_ID": dbUser.FamilyID})

		resp := map[string]interface{}{
			"user":    dbUser,
			"private": private,
			"family":  family,
		}
		if len(families) > 0 {
			resp["family_info"] = families[0]
		}
		json.NewEncoder(w).Encode(resp)

	case http.MethodPost:
		var newTask internal.Task
		json.NewDecoder(r.Body).Decode(&newTask)
		newTask.UserID = dbUser.UserID
		if newTask.Scope == "family" {
			fID := dbUser.FamilyID
			newTask.FamilyID = &fID
		}
		h.SB.InsertInto("tasks", newTask)
		w.WriteHeader(201)

	case http.MethodPatch:
		idStr := r.URL.Query().Get("id")
		var updateData map[string]interface{}
		json.NewDecoder(r.Body).Decode(&updateData)
		taskID, _ := strconv.ParseInt(idStr, 10, 64)
		h.SB.UpdateTable("tasks", updateData, map[string]interface{}{"task_ID": taskID})
		w.WriteHeader(200)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		taskID, _ := strconv.ParseInt(idStr, 10, 64)
		h.SB.DeleteFrom("tasks", map[string]interface{}{"task_ID": taskID})
		w.WriteHeader(200)
	}
}

func (h *Handler) JoinFamilyHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	users, _ := h.SB.SelectFrom("users", "user_ID,username,password,family_ID", map[string]interface{}{"username": username})
	if len(users) == 0 {
		http.Error(w, "User not found", 404)
		return
	}

	userMap := users[0]
	var user internal.User
	if uIDFloat, ok := userMap["user_ID"].(float64); ok {
		user.UserID = int64(uIDFloat)
	}
	if famID, ok := userMap["family_ID"].(float64); ok {
		user.FamilyID = int64(famID)
	}
	oldFamilyID := user.FamilyID

	families, _ := h.SB.SelectFrom("families", "*", map[string]interface{}{"code": body.Code})
	if len(families) == 0 {
		http.Error(w, "Invalid code", 404)
		return
	}

	targetMap := families[0]
	var target internal.Family
	if fID, ok := targetMap["family_ID"].(float64); ok {
		target.FamilyID = int64(fID)
	}
	if members, ok := targetMap["members"].([]interface{}); ok {
		for _, m := range members {
			if s, ok := m.(string); ok {
				target.Members = append(target.Members, s)
			}
		}
	}

	h.SB.UpdateTable("users", map[string]interface{}{"family_ID": target.FamilyID}, map[string]interface{}{"username": username})

	newMembers := append(target.Members, username)
	h.SB.UpdateTable("families", map[string]interface{}{"members": newMembers}, map[string]interface{}{"family_ID": target.FamilyID})

	oldFamilies, _ := h.SB.SelectFrom("families", "*", map[string]interface{}{"family_ID": oldFamilyID})

	if len(oldFamilies) > 0 {
		oldFamMap := oldFamilies[0]
		var oldFam internal.Family
		if members, ok := oldFamMap["members"].([]interface{}); ok {
			for _, m := range members {
				if s, ok := m.(string); ok {
					oldFam.Members = append(oldFam.Members, s)
				}
			}
		}

		var remainingMembers []string
		for _, m := range oldFam.Members {
			if m != username {
				remainingMembers = append(remainingMembers, m)
			}
		}

		if len(remainingMembers) == 0 {
			h.SB.DeleteFrom("tasks", map[string]interface{}{"family_ID": oldFamilyID})
			h.SB.DeleteFrom("families", map[string]interface{}{"family_ID": oldFamilyID})
			log.Printf("Cleaned up orphaned family %d after user %s joined a new one", oldFamilyID, username)
		} else {
			h.SB.UpdateTable("families", map[string]interface{}{"members": remainingMembers}, map[string]interface{}{"family_ID": oldFamilyID})
		}
	}

	w.WriteHeader(200)
}

func (h *Handler) LeaveFamilyHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")

	users, _ := h.SB.SelectFrom("users", "user_ID,username,password,family_ID", map[string]interface{}{"username": username})
	if len(users) == 0 {
		http.Error(w, "User not found", 404)
		return
	}

	userMap := users[0]
	var user internal.User
	if uID, ok := userMap["user_ID"].(float64); ok {
		user.UserID = int64(uID)
	}
	if famID, ok := userMap["family_ID"].(float64); ok {
		user.FamilyID = int64(famID)
	}

	families, _ := h.SB.SelectFrom("families", "*", map[string]interface{}{"family_ID": user.FamilyID})

	if len(families) > 0 {
		oldFamMap := families[0]
		var oldFam internal.Family
		if members, ok := oldFamMap["members"].([]interface{}); ok {
			for _, m := range members {
				if s, ok := m.(string); ok {
					oldFam.Members = append(oldFam.Members, s)
				}
			}
		}

		var updatedMembers []string

		for _, m := range oldFam.Members {
			if m != username {
				updatedMembers = append(updatedMembers, m)
			}
		}

		if len(updatedMembers) == 0 {
			h.SB.DeleteFrom("tasks", map[string]interface{}{"family_ID": user.FamilyID})
			h.SB.DeleteFrom("families", map[string]interface{}{"family_ID": user.FamilyID})
			log.Printf("Family %d deleted: no members left", user.FamilyID)
		} else {
			h.SB.UpdateTable("families", map[string]interface{}{"members": updatedMembers}, map[string]interface{}{"family_ID": user.FamilyID})
		}
	}
	newFID := h.getNewFamilyID()
	newFam := internal.Family{
		FamilyID: newFID,
		OwnerID:  user.UserID,
		Members:  []string{username},
		Code:     generateShortCode(7),
	}

	h.SB.InsertInto("families", newFam)

	h.SB.UpdateTable("users", map[string]interface{}{
		"family_ID": newFID,
	}, map[string]interface{}{"username": username})

	w.WriteHeader(200)
}
