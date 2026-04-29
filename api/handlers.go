package api

import (
	"encoding/json"
	"log"
	"math/rand"
	"mon-projet/internal"
	"net/http"
	"strconv"
	"time"

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
			log.Printf("comparing family_ID: %d with current max_ID %d", f.FamilyID, maxID)
			if f.FamilyID > maxID {
				maxID = f.FamilyID
			}
		}
	} else if err != nil {
		log.Printf("Error retrieving IDs: %v", err)
	}
	log.Printf("Max id found is %d, and we will use %d as an id", maxID, maxID+1)
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
			log.Printf("comparing user_ID: %d with current max_ID %d", f.UserID, maxID)
			if f.UserID > maxID {
				maxID = f.UserID
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

	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", creds.User).Execute(&users)

	var user internal.User
	if len(users) == 0 {

		new_family_ID := h.getNewFamilyID()
		new_user_ID := h.getNewUserID()

		log.Printf("Account creation: %s", creds.User)
		user = internal.User{
			UserID:   new_user_ID,
			Username: creds.User,
			Password: creds.Pass,
			FamilyID: new_family_ID,
		}

		var inserted []internal.User
		err := h.SB.DB.From("users").Insert(user).Execute(&inserted)
		if err != nil || len(inserted) == 0 {
			log.Printf("User insertion error: %v", err)
			http.Error(w, "Creation error", 500)
			return
		}
		user = inserted[0]

		var existingCodes []struct{ Code string }
		err = h.SB.DB.From("families").Select("code").Execute(&existingCodes)
		if err != nil {
			log.Printf("Warning: unable to verify uniqueness via full list: %v", err)
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

		// 3. Insert family into Supabase
		var insertedFamilies []internal.Family
		errFam := h.SB.DB.From("families").Insert(newFamily).Execute(&insertedFamilies)

		if errFam == nil && len(insertedFamilies) > 0 {
			createdFamily := insertedFamilies[0]
			user.FamilyID = createdFamily.FamilyID

			// 4. Update user to link them to their new family
			uIDStr := strconv.FormatInt(user.UserID, 10)
			h.SB.DB.From("users").Update(map[string]interface{}{
				"family_ID": user.FamilyID,
			}).Eq("user_ID", uIDStr).Execute(nil)

			log.Printf("✅ Family #%d created for %s", user.FamilyID, user.Username)
		} else {
			log.Printf("⚠️ Family creation error: %v", errFam)
		}

		// --- END FAMILY CREATION ---

	} else {
		user = users[0]
		if user.Password != creds.Pass {
			http.Error(w, "Incorrect password", 401)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) TasksHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "Missing user", 400)
		return
	}

	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", username).Execute(&users)
	if len(users) == 0 {
		http.Error(w, "User not found", 404)
		return
	}
	dbUser := users[0]

	switch r.Method {
	case http.MethodGet:
		var private, family []internal.Task
		var families []internal.Family
		fID := strconv.FormatInt(dbUser.FamilyID, 10)

		h.SB.DB.From("tasks").Select("*").Eq("user_ID", strconv.FormatInt(dbUser.UserID, 10)).Eq("scope", "private").Execute(&private)
		h.SB.DB.From("tasks").Select("*").Eq("family_ID", fID).Eq("scope", "family").Execute(&family)
		h.SB.DB.From("families").Select("*").Eq("family_ID", fID).Execute(&families)

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
		h.SB.DB.From("tasks").Insert(newTask).Execute(nil)
		w.WriteHeader(201)

	case http.MethodPatch:
		idStr := r.URL.Query().Get("id")
		var updateData map[string]interface{}
		json.NewDecoder(r.Body).Decode(&updateData)
		h.SB.DB.From("tasks").Update(updateData).Eq("task_ID", idStr).Execute(nil)
		w.WriteHeader(200)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		h.SB.DB.From("tasks").Delete().Eq("task_ID", idStr).Execute(nil)
		w.WriteHeader(200)
	}
}

func (h *Handler) JoinFamilyHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// 1. Get the user and their current family to clean up later
	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", username).Execute(&users)
	if len(users) == 0 {
		http.Error(w, "User not found", 404)
		return
	}
	user := users[0]
	oldFamilyID := user.FamilyID

	// 2. Find the target family they want to join
	var families []internal.Family
	h.SB.DB.From("families").Select("*").Eq("code", body.Code).Execute(&families)
	if len(families) == 0 {
		http.Error(w, "Invalid code", 404)
		return
	}
	target := families[0]

	// 3. Update the user's family_ID in the database
	h.SB.DB.From("users").Update(map[string]interface{}{"family_ID": target.FamilyID}).Eq("username", username).Execute(nil)

	// 4. Update the target family's member list
	newMembers := append(target.Members, username)
	h.SB.DB.From("families").Update(map[string]interface{}{"members": newMembers}).Eq("family_ID", strconv.FormatInt(target.FamilyID, 10)).Execute(nil)

	// 5. CLEAN UP: Check if the old family is now empty and delete it
	var oldFamilies []internal.Family
	fIDStr := strconv.FormatInt(oldFamilyID, 10)
	h.SB.DB.From("families").Select("*").Eq("family_ID", fIDStr).Execute(&oldFamilies)

	if len(oldFamilies) > 0 {
		oldFam := oldFamilies[0]
		var remainingMembers []string
		for _, m := range oldFam.Members {
			if m != username {
				remainingMembers = append(remainingMembers, m)
			}
		}

		if len(remainingMembers) == 0 {
			// Delete tasks and family if no one is left
			h.SB.DB.From("tasks").Delete().Eq("family_ID", fIDStr).Execute(nil)
			h.SB.DB.From("families").Delete().Eq("family_ID", fIDStr).Execute(nil)
			log.Printf("Cleaned up orphaned family %s after user %s joined a new one", fIDStr, username)
		} else {
			// Otherwise, just update the old family's member list
			h.SB.DB.From("families").Update(map[string]interface{}{"members": remainingMembers}).Eq("family_ID", fIDStr).Execute(nil)
		}
	}

	w.WriteHeader(200)
}

func (h *Handler) LeaveFamilyHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")

	// 1. Get user data
	var users []internal.User
	h.SB.DB.From("users").Select("*").Eq("username", username).Execute(&users)
	if len(users) == 0 {
		http.Error(w, "User not found", 404)
		return
	}
	user := users[0]

	// 2. Get current family data
	var families []internal.Family
	fIDStr := strconv.FormatInt(user.FamilyID, 10)
	h.SB.DB.From("families").Select("*").Eq("family_ID", fIDStr).Execute(&families)

	if len(families) > 0 {
		oldFam := families[0]
		var updatedMembers []string

		// Remove the user from the members list
		for _, m := range oldFam.Members {
			if m != username {
				updatedMembers = append(updatedMembers, m)
			}
		}

		// 3. Check if family is now empty
		if len(updatedMembers) == 0 {
			// Delete family tasks first (to avoid foreign key conflicts)
			h.SB.DB.From("tasks").Delete().Eq("family_ID", fIDStr).Execute(nil)
			// Delete the family
			h.SB.DB.From("families").Delete().Eq("family_ID", fIDStr).Execute(nil)
			log.Printf("Family %s deleted: no members left", fIDStr)
		} else {
			// Update the family with the remaining members
			h.SB.DB.From("families").Update(map[string]interface{}{"members": updatedMembers}).Eq("family_ID", fIDStr).Execute(nil)
		}
	}

	// 4. Create a brand new family for the user
	newFID := h.getNewFamilyID()
	newFam := internal.Family{
		FamilyID: newFID,
		OwnerID:  user.UserID,
		Members:  []string{username},
		Code:     generateShortCode(7),
	}

	// Insert new family
	h.SB.DB.From("families").Insert(newFam).Execute(nil)

	// 5. Update the user to link them to this new family ID
	h.SB.DB.From("users").Update(map[string]interface{}{
		"family_ID": newFID,
	}).Eq("username", username).Execute(nil)

	w.WriteHeader(200)
}
