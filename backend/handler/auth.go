package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"openmanage/backend/auth"
	"openmanage/backend/middleware"
	"openmanage/backend/model"
)

type AuthHandler struct {
	Store     *auth.Store
	JWTSecret []byte
}

func NewAuthHandler(jwtSecret string) *AuthHandler {
	store, err := auth.NewStore()
	if err != nil {
		panic("failed to create auth store: " + err.Error())
	}
	return &AuthHandler{
		Store:     store,
		JWTSecret: []byte(jwtSecret),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if !h.Store.Verify(req.Username, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.generateToken(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Set HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("HTTPS") == "true",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	writeJSON(w, http.StatusOK, model.LoginResponse{Token: token})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// User info is extracted from context by middleware
	userID := r.Context().Value(middleware.UserIDKey)
	if userID == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": userID.(string)})
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": false})
		return
	}

	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.JWTSecret, nil
	})

	if err != nil || !token.Valid {
		writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": false})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"username":      claims["sub"],
	})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req model.ChangePasswordRequest
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.Store.UpdatePassword(req.OldPassword, req.NewPassword); err != nil {
		if err == auth.ErrInvalidPassword {
			writeError(w, http.StatusUnauthorized, "invalid old password")
			return
		}
		if err == auth.ErrPasswordTooShort {
			writeError(w, http.StatusBadRequest, "new password must be at least 6 characters")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}

func (h *AuthHandler) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.JWTSecret)
}
