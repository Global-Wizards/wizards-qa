package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Global-Wizards/wizards-qa/web/backend/auth"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
)

// requireProjectAccess checks that the current user is a member of the project.
// If requiredRoles is non-empty, the user must have one of those roles.
// Returns true if access is granted; writes an error response and returns false otherwise.
func (s *Server) requireProjectAccess(w http.ResponseWriter, r *http.Request, projectID string, requiredRoles ...string) bool {
	claims := auth.UserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return false
	}
	// Global admins always have access
	if claims.Role == "admin" {
		return true
	}
	role, err := s.store.GetProjectMemberRole(projectID, claims.UserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "Not a member of this project")
		return false
	}
	if len(requiredRoles) == 0 {
		return true
	}
	for _, rr := range requiredRoles {
		if role == rr {
			return true
		}
	}
	respondError(w, http.StatusForbidden, "Insufficient project permissions")
	return false
}

// --- Project CRUD ---

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list projects")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"projects": nonNil(projects),
		"total":    len(projects),
	})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var p store.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if p.Name == "" {
		respondError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	now := time.Now().Format(time.RFC3339)
	p.ID = newID("proj")
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Color == "" {
		p.Color = "#6366f1"
	}
	if p.Icon == "" {
		p.Icon = "gamepad-2"
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.Settings == nil {
		p.Settings = map[string]string{}
	}

	if claims := auth.UserFromContext(r.Context()); claims != nil {
		p.CreatedBy = claims.UserID
	}

	if p.CreatedBy != "" {
		// Atomically create project and add creator as owner
		if err := s.store.CreateProjectWithOwner(p, store.ProjectMember{
			ID:        newID("pm"),
			ProjectID: p.ID,
			UserID:    p.CreatedBy,
			Role:      "owner",
			CreatedAt: now,
		}); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create project")
			return
		}
	} else {
		if err := s.store.SaveProject(p); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create project")
			return
		}
	}

	respondJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id) {
		return
	}
	p, err := s.store.GetProject(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	respondJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id, "owner") {
		return
	}

	existing, err := s.store.GetProject(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	var updates store.Project
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply updates
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.GameURL != "" {
		existing.GameURL = updates.GameURL
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Color != "" {
		existing.Color = updates.Color
	}
	if updates.Icon != "" {
		existing.Icon = updates.Icon
	}
	if updates.Tags != nil {
		existing.Tags = updates.Tags
	}
	if updates.Settings != nil {
		existing.Settings = updates.Settings
	}
	existing.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.store.UpdateProject(*existing); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update project")
		return
	}

	respondJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id, "owner") {
		return
	}
	if err := s.store.DeleteProject(id); err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Project Stats ---

func (s *Server) handleGetProjectStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id) {
		return
	}
	stats, err := s.store.GetStatsByProject(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get project stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// --- Project-scoped entity lists ---

func (s *Server) handleListProjectAnalyses(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id) {
		return
	}
	analyses, err := s.store.ListAnalysesByProject(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list project analyses")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"analyses": nonNil(analyses),
		"total":    len(analyses),
	})
}

func (s *Server) handleListProjectTestPlans(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id) {
		return
	}
	plans, err := s.store.ListTestPlansByProject(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list project test plans")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"plans": nonNil(plans),
		"total": len(plans),
	})
}

func (s *Server) handleListProjectTests(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id) {
		return
	}
	tests, err := s.store.ListTestResultsByProject(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list project tests")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tests": nonNil(tests),
		"total": len(tests),
	})
}

// --- Project Members ---

func (s *Server) handleListProjectMembers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, id) {
		return
	}
	members, err := s.store.ListProjectMembers(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list project members")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"members": nonNil(members),
		"total":   len(members),
	})
}

func (s *Server) handleAddProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, projectID, "owner") {
		return
	}

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	user, err := s.store.GetUserByEmail(req.Email)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found with that email")
		return
	}

	role := req.Role
	if role == "" {
		role = "member"
	}

	now := time.Now().Format(time.RFC3339)
	member := store.ProjectMember{
		ID:        newID("pm"),
		ProjectID: projectID,
		UserID:    user.ID,
		Role:      role,
		CreatedAt: now,
	}

	if err := s.store.AddProjectMember(member); err != nil {
		respondError(w, http.StatusConflict, "User is already a member of this project")
		return
	}

	member.Email = user.Email
	member.DisplayName = user.DisplayName
	respondJSON(w, http.StatusCreated, member)
}

func (s *Server) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, projectID, "owner") {
		return
	}
	userID := chi.URLParam(r, "userId")

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.store.UpdateProjectMemberRole(projectID, userID, req.Role); err != nil {
		respondError(w, http.StatusNotFound, "Member not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleRemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if !s.requireProjectAccess(w, r, projectID, "owner") {
		return
	}
	userID := chi.URLParam(r, "userId")

	if err := s.store.RemoveProjectMember(projectID, userID); err != nil {
		respondError(w, http.StatusNotFound, "Member not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
