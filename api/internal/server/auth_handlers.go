package server

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/codearena/api/internal/auth"
    "golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type authResponse struct {
    Token  string `json:"token"`
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
}

func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        methodNotAllowed(w)
        return
    }

    var req registerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    req.Email = strings.TrimSpace(strings.ToLower(req.Email))
    if req.Email == "" || req.Password == "" {
        http.Error(w, "email and password are required", http.StatusBadRequest)
        return
    }
    if len(req.Password) < 8 {
        http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
        return
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    userID, err := s.store.CreateUser(r.Context(), req.Email, string(hash))
    if err != nil {
        if strings.Contains(err.Error(), "already registered") || strings.Contains(err.Error(), "unique") {
            http.Error(w, "email already registered", http.StatusConflict)
            return
        }
        http.Error(w, "failed to create user", http.StatusInternalServerError)
        return
    }

    token, err := auth.Sign(userID, req.Email, "user")
    if err != nil {
        http.Error(w, "failed to issue token", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(authResponse{Token: token, UserID: userID, Email: req.Email, Role: "user"})
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        methodNotAllowed(w)
        return
    }

    var req registerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    req.Email = strings.TrimSpace(strings.ToLower(req.Email))
    if req.Email == "" || req.Password == "" {
        http.Error(w, "email and password are required", http.StatusBadRequest)
        return
    }

    user, err := s.store.GetUserByEmail(r.Context(), req.Email)
    if err != nil {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    token, err := auth.Sign(user.ID, user.Email, user.Role)
    if err != nil {
        http.Error(w, "failed to issue token", http.StatusInternalServerError)
        return
    }

    respondJSON(w, authResponse{Token: token, UserID: user.ID, Email: user.Email, Role: user.Role})
}
