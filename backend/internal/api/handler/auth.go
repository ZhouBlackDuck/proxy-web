package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/store"
)

var jwtSecret = []byte("proxy-web-secret-key-change-in-production")

type AuthHandler struct {
	cfg   *config.Config
	store *store.FileStore
}

func NewAuthHandler(cfg *config.Config, store *store.FileStore) *AuthHandler {
	return &AuthHandler{cfg: cfg, store: store}
}

type loginRequest struct {
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"`
}

// Setup handles initial password setup (first time only)
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	if h.cfg.PasswordHash != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "password already set",
		})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if len(req.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "password must be at least 6 characters",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to hash password",
		})
		return
	}

	h.cfg.PasswordHash = string(hash)
	if err := h.cfg.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save settings",
		})
		return
	}

	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to generate token",
		})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresIn: 86400,
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.cfg.PasswordHash == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "setup required",
		})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid password",
		})
		return
	}

	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to generate token",
		})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresIn: 86400,
	})
}

// Status returns auth status (whether setup is needed or already configured)
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"configured": h.cfg.PasswordHash != "",
	})
}

// Check validates the current token
func (h *AuthHandler) Check(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"valid": true})
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.PasswordHash), []byte(req.OldPassword)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid old password",
		})
		return
	}

	if len(req.NewPassword) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "password must be at least 6 characters",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to hash password",
		})
		return
	}

	h.cfg.PasswordHash = string(hash)
	if err := h.cfg.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save settings",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}

func generateToken() (string, error) {
	claims := jwt.MapClaims{
		"sub": "user",
		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
