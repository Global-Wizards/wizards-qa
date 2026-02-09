package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Global-Wizards/wizards-qa/web/backend/auth"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// authTokenResponse builds the standard token+user response for auth endpoints.
func authTokenResponse(user *store.User, accessToken, refreshToken string) map[string]interface{} {
	return map[string]interface{}{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"displayName": user.DisplayName,
			"role":        user.Role,
		},
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if !emailRegex.MatchString(req.Email) {
		respondError(w, http.StatusBadRequest, "Invalid email format")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}
	if req.DisplayName == "" {
		respondError(w, http.StatusBadRequest, "Display name is required")
		return
	}

	// Check if email already exists
	if existing, _ := s.store.GetUserByEmail(req.Email); existing != nil {
		respondError(w, http.StatusConflict, "Email already registered")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// First user becomes admin
	role := "member"
	count, _ := s.store.UserCount()
	if count == 0 {
		role = "admin"
	}

	user := store.User{
		ID:           newID("user"),
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().Format(time.RFC3339),
	}

	if err := s.store.CreateUser(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	accessToken, refreshToken, err := auth.GenerateTokens(&user, s.jwtSecret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	respondJSON(w, http.StatusCreated, authTokenResponse(&user, accessToken, refreshToken))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := s.store.GetUserByEmail(req.Email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	accessToken, refreshToken, err := auth.GenerateTokens(user, s.jwtSecret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	respondJSON(w, http.StatusOK, authTokenResponse(user, accessToken, refreshToken))
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims, err := auth.ValidateRefreshToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	user, err := s.store.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "User not found")
		return
	}

	accessToken, refreshToken, err := auth.GenerateTokens(user, s.jwtSecret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	respondJSON(w, http.StatusOK, authTokenResponse(user, accessToken, refreshToken))
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.UserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	user, err := s.store.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":          user.ID,
		"email":       user.Email,
		"displayName": user.DisplayName,
		"role":        user.Role,
		"createdAt":   user.CreatedAt,
	})
}
