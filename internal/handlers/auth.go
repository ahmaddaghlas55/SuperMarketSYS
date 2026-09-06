package handlers

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"

	"supermarket/internal/middleware"
	"supermarket/internal/services"
)

type AuthHandler struct{ auth *services.AuthService }

func NewAuthHandler(auth *services.AuthService) *AuthHandler { return &AuthHandler{auth: auth} }

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers(r.Context())
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in userRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	user, err := h.auth.CreateUser(r.Context(), currentUserID(r), in.Username, in.Password, in.Role)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_id")
		return
	}
	var in userRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err = h.auth.UpdateUser(r.Context(), currentUserID(r), id, in.Username, in.Password, in.Role); err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AuthHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_id")
		return
	}
	if err = h.auth.SetUserActive(r.Context(), currentUserID(r), id, false); err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active": false})
}

func (h *AuthHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_id")
		return
	}
	if err = h.auth.SetUserActive(r.Context(), currentUserID(r), id, true); err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active": true})
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Username == "" || len(request.Password) < 8 {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	user, err := h.auth.Bootstrap(r.Context(), request.Username, request.Password)
	if err != nil {
		if errors.Is(err, services.ErrBootstrapDisabled) {
			middleware.WriteError(w, http.StatusConflict, "bootstrap_disabled")
			return
		}
		middleware.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, token, expiresAt, err := h.auth.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		middleware.WriteError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token, "expires_at": expiresAt, "user": user,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		middleware.WriteError(w, http.StatusUnauthorized, "authentication_required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	header := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if err := h.auth.Logout(r.Context(), strings.TrimSpace(header)); err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "logout_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_content_type")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "invalid_json")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
