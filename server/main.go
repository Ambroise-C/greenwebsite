package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Server struct {
	db       *sql.DB
	sessions map[string]string
	mu       sync.RWMutex
}

type APIError struct {
	Error string `json:"error"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string    `json:"token"`
	State AppState  `json:"state"`
}

type TaskRequest struct {
	Scope string `json:"scope"`
	Title string `json:"title"`
}

type TaskMutationRequest struct {
	Scope string `json:"scope"`
	ID    int64  `json:"id"`
}

type JoinFamilyRequest struct {
	Code string `json:"code"`
}

type KickMemberRequest struct {
	Username string `json:"username"`
}

type UserInfo struct {
	Username string `json:"username"`
	Code     string `json:"code"`
	FamilyID string `json:"familyId"`
}

type FamilyInfo struct {
	Owner   string   `json:"owner"`
	Members []string `json:"members"`
}

type Task struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Completed   bool    `json:"completed"`
	CompletedBy *string `json:"completedBy,omitempty"`
}

type AppState struct {
	User         UserInfo    `json:"user"`
	Family       FamilyInfo  `json:"family"`
	PrivateTasks []Task      `json:"privateTasks"`
	FamilyTasks  []Task      `json:"familyTasks"`
}

func main() {
	if err := os.MkdirAll("data", 0o755); err != nil {
		log.Fatalf("cannot create data directory: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join("data", "leaftask.db"))
	if err != nil {
		log.Fatalf("cannot open database: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("cannot initialize database: %v", err)
	}

	srv := &Server{db: db, sessions: make(map[string]string)}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/auth", srv.handleAuth)
	mux.HandleFunc("/api/logout", srv.withAuth(srv.handleLogout))
	mux.HandleFunc("/api/state", srv.withAuth(srv.handleState))
	mux.HandleFunc("/api/tasks", srv.withAuth(srv.handleCreateTask))
	mux.HandleFunc("/api/tasks/toggle", srv.withAuth(srv.handleToggleTask))
	mux.HandleFunc("/api/tasks/delete", srv.withAuth(srv.handleDeleteTask))
	mux.HandleFunc("/api/family/join", srv.withAuth(srv.handleJoinFamily))
	mux.HandleFunc("/api/family/leave", srv.withAuth(srv.handleLeaveFamily))
	mux.HandleFunc("/api/family/kick", srv.withAuth(srv.handleKickMember))

	staticFS := http.FileServer(http.Dir(".."))
	mux.Handle("/", staticFS)

	addr := ":8080"
	log.Printf("LeafTask server running on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func initDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password_hash TEXT NOT NULL,
		code TEXT NOT NULL UNIQUE,
		family_id TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS families (
		id TEXT PRIMARY KEY,
		owner_username TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS family_members (
		family_id TEXT NOT NULL,
		username TEXT NOT NULL,
		PRIMARY KEY (family_id, username),
		FOREIGN KEY (family_id) REFERENCES families(id),
		FOREIGN KEY (username) REFERENCES users(username)
	);

	CREATE TABLE IF NOT EXISTS private_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		title TEXT NOT NULL,
		completed INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (username) REFERENCES users(username)
	);

	CREATE TABLE IF NOT EXISTS family_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		family_id TEXT NOT NULL,
		title TEXT NOT NULL,
		completed INTEGER NOT NULL DEFAULT 0,
		completed_by TEXT,
		FOREIGN KEY (family_id) REFERENCES families(id)
	);
	`

	_, err := db.Exec(query)
	return err
}

func (s *Server) withAuth(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, err := s.authUser(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, APIError{Error: "unauthorized"})
			return
		}
		next(w, r, username)
	}
}

func (s *Server) authUser(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", errors.New("missing token")
	}
	token := strings.TrimPrefix(auth, "Bearer ")

	s.mu.RLock()
	username, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return "", errors.New("invalid session")
	}
	return username, nil
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid request"})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "username and password are required"})
		return
	}

	if err := s.ensureUser(username, password); err != nil {
		writeJSON(w, http.StatusUnauthorized, APIError{Error: err.Error()})
		return
	}

	token, err := randomToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot create session"})
		return
	}

	s.mu.Lock()
	s.sessions[token] = username
	s.mu.Unlock()

	state, err := s.buildState(username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load state"})
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{Token: token, State: state})
}

func (s *Server) ensureUser(username, password string) error {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	if err == nil {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			return errors.New("MDP incorrect")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	code, err := s.uniqueCode()
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec("INSERT INTO users(username, password_hash, code, family_id) VALUES(?, ?, ?, ?)", username, string(passHash), code, username); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO families(id, owner_username) VALUES(?, ?)", username, username); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO family_members(family_id, username) VALUES(?, ?)", username, username); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Server) uniqueCode() (string, error) {
	for i := 0; i < 30; i++ {
		code, err := randomCode(6)
		if err != nil {
			return "", err
		}
		var exists int
		err = s.db.QueryRow("SELECT 1 FROM users WHERE code = ?", code).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return code, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
	}
	return "", errors.New("cannot generate unique code")
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	_ = username
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")

	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	state, err := s.buildState(username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load state"})
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) buildState(username string) (AppState, error) {
	var state AppState
	var user UserInfo
	if err := s.db.QueryRow("SELECT username, code, family_id FROM users WHERE username = ?", username).Scan(&user.Username, &user.Code, &user.FamilyID); err != nil {
		return state, err
	}
	state.User = user

	var owner string
	if err := s.db.QueryRow("SELECT owner_username FROM families WHERE id = ?", user.FamilyID).Scan(&owner); err != nil {
		return state, err
	}
	state.Family.Owner = owner

	rows, err := s.db.Query("SELECT username FROM family_members WHERE family_id = ? ORDER BY username", user.FamilyID)
	if err != nil {
		return state, err
	}
	defer rows.Close()

	for rows.Next() {
		var member string
		if err := rows.Scan(&member); err != nil {
			return state, err
		}
		state.Family.Members = append(state.Family.Members, member)
	}
	if err := rows.Err(); err != nil {
		return state, err
	}

	privateRows, err := s.db.Query("SELECT id, title, completed FROM private_tasks WHERE username = ? ORDER BY id DESC", username)
	if err != nil {
		return state, err
	}
	defer privateRows.Close()

	for privateRows.Next() {
		var t Task
		var completed int
		if err := privateRows.Scan(&t.ID, &t.Title, &completed); err != nil {
			return state, err
		}
		t.Completed = completed == 1
		state.PrivateTasks = append(state.PrivateTasks, t)
	}
	if err := privateRows.Err(); err != nil {
		return state, err
	}

	familyRows, err := s.db.Query("SELECT id, title, completed, completed_by FROM family_tasks WHERE family_id = ? ORDER BY id DESC", user.FamilyID)
	if err != nil {
		return state, err
	}
	defer familyRows.Close()

	for familyRows.Next() {
		var t Task
		var completed int
		var by sql.NullString
		if err := familyRows.Scan(&t.ID, &t.Title, &completed, &by); err != nil {
			return state, err
		}
		t.Completed = completed == 1
		if by.Valid {
			t.CompletedBy = &by.String
		}
		state.FamilyTasks = append(state.FamilyTasks, t)
	}

	return state, familyRows.Err()
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid request"})
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "title required"})
		return
	}

	scope := strings.TrimSpace(req.Scope)
	switch scope {
	case "private":
		if _, err := s.db.Exec("INSERT INTO private_tasks(username, title, completed) VALUES(?, ?, 0)", username, title); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot create task"})
			return
		}
	case "family":
		var familyID string
		if err := s.db.QueryRow("SELECT family_id FROM users WHERE username = ?", username).Scan(&familyID); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load family"})
			return
		}
		if _, err := s.db.Exec("INSERT INTO family_tasks(family_id, title, completed, completed_by) VALUES(?, ?, 0, NULL)", familyID, title); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot create task"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid scope"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleToggleTask(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req TaskMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid request"})
		return
	}

	scope := strings.TrimSpace(req.Scope)
	switch scope {
	case "private":
		if _, err := s.db.Exec(`
			UPDATE private_tasks
			SET completed = CASE completed WHEN 1 THEN 0 ELSE 1 END
			WHERE id = ? AND username = ?
		`, req.ID, username); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot toggle task"})
			return
		}
	case "family":
		var familyID string
		if err := s.db.QueryRow("SELECT family_id FROM users WHERE username = ?", username).Scan(&familyID); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load family"})
			return
		}
		if _, err := s.db.Exec(`
			UPDATE family_tasks
			SET completed = CASE completed WHEN 1 THEN 0 ELSE 1 END,
				completed_by = CASE completed WHEN 1 THEN NULL ELSE ? END
			WHERE id = ? AND family_id = ?
		`, username, req.ID, familyID); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot toggle task"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid scope"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req TaskMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid request"})
		return
	}

	scope := strings.TrimSpace(req.Scope)
	switch scope {
	case "private":
		if _, err := s.db.Exec("DELETE FROM private_tasks WHERE id = ? AND username = ?", req.ID, username); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot delete task"})
			return
		}
	case "family":
		var familyID string
		if err := s.db.QueryRow("SELECT family_id FROM users WHERE username = ?", username).Scan(&familyID); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load family"})
			return
		}
		if _, err := s.db.Exec("DELETE FROM family_tasks WHERE id = ? AND family_id = ?", req.ID, familyID); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot delete task"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid scope"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleJoinFamily(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req JoinFamilyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid request"})
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "code required"})
		return
	}

	var targetOwner string
	if err := s.db.QueryRow("SELECT username FROM users WHERE code = ?", code).Scan(&targetOwner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusBadRequest, APIError{Error: "Code invalide"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot find family"})
		return
	}
	if targetOwner == username {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "Code invalide"})
		return
	}

	if err := s.moveUserToFamily(username, targetOwner); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot join family"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) moveUserToFamily(username, targetFamilyID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldFamilyID string
	if err := tx.QueryRow("SELECT family_id FROM users WHERE username = ?", username).Scan(&oldFamilyID); err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM family_members WHERE family_id = ? AND username = ?", oldFamilyID, username); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE users SET family_id = ? WHERE username = ?", targetFamilyID, username); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT OR IGNORE INTO family_members(family_id, username) VALUES(?, ?)", targetFamilyID, username); err != nil {
		return err
	}

	if oldFamilyID == username {
		if _, err := tx.Exec("DELETE FROM family_tasks WHERE family_id = ?", oldFamilyID); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM family_members WHERE family_id = ?", oldFamilyID); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM families WHERE id = ?", oldFamilyID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Server) handleLeaveFamily(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot leave family"})
		return
	}
	defer tx.Rollback()

	var familyID string
	if err := tx.QueryRow("SELECT family_id FROM users WHERE username = ?", username).Scan(&familyID); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot leave family"})
		return
	}

	if familyID == username {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	if _, err := tx.Exec("DELETE FROM family_members WHERE family_id = ? AND username = ?", familyID, username); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot leave family"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET family_id = ? WHERE username = ?", username, username); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot leave family"})
		return
	}

	if _, err := tx.Exec("INSERT OR IGNORE INTO families(id, owner_username) VALUES(?, ?)", username, username); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot create personal family"})
		return
	}
	if _, err := tx.Exec("INSERT OR IGNORE INTO family_members(family_id, username) VALUES(?, ?)", username, username); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot create personal family"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot leave family"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleKickMember(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req KickMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid request"})
		return
	}
	member := strings.TrimSpace(req.Username)
	if member == "" || member == username {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid member"})
		return
	}

	var familyID string
	if err := s.db.QueryRow("SELECT family_id FROM users WHERE username = ?", username).Scan(&familyID); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load family"})
		return
	}

	var owner string
	if err := s.db.QueryRow("SELECT owner_username FROM families WHERE id = ?", familyID).Scan(&owner); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot load family"})
		return
	}

	if owner != username {
		writeJSON(w, http.StatusForbidden, APIError{Error: "only owner can kick"})
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot kick member"})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM family_members WHERE family_id = ? AND username = ?", familyID, member); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot kick member"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET family_id = ? WHERE username = ?", member, member); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot kick member"})
		return
	}
	if _, err := tx.Exec("INSERT OR IGNORE INTO families(id, owner_username) VALUES(?, ?)", member, member); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot rebuild member family"})
		return
	}
	if _, err := tx.Exec("INSERT OR IGNORE INTO family_members(family_id, username) VALUES(?, ?)", member, member); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot rebuild member family"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: "cannot kick member"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func randomToken(size int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}

func randomCode(size int) (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("json encode error: %v\n", err)
	}
}
