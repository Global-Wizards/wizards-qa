package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Global-Wizards/wizards-qa/web/backend/auth"
	"github.com/Global-Wizards/wizards-qa/web/backend/store"
)

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
	p.ID = fmt.Sprintf("proj-%d", time.Now().UnixNano())
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

	if err := s.store.SaveProject(p); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create project")
		return
	}

	// Auto-add creator as owner member
	if p.CreatedBy != "" {
		s.store.AddProjectMember(store.ProjectMember{
			ID:        fmt.Sprintf("pm-%d", time.Now().UnixNano()),
			ProjectID: p.ID,
			UserID:    p.CreatedBy,
			Role:      "owner",
			CreatedAt: now,
		})
	}

	respondJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	p, err := s.store.GetProject(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	respondJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")

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
	if err := s.store.DeleteProject(id); err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Project Stats ---

func (s *Server) handleGetProjectStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
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
		ID:        fmt.Sprintf("pm-%d", time.Now().UnixNano()),
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
	userID := chi.URLParam(r, "userId")

	if err := s.store.RemoveProjectMember(projectID, userID); err != nil {
		respondError(w, http.StatusNotFound, "Member not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
