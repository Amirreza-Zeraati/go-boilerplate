// Package handler contains the Gin HTTP handlers. Handlers translate between
// HTTP and the service layer: bind+validate input, call a service, map domain
// errors to status codes, and write a response. No business logic lives here.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Amirreza-Zeraati/go-boilerplate/internal/config"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/dto"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/middleware"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/response"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/service"
	"github.com/Amirreza-Zeraati/go-boilerplate/internal/session"
)

// AuthHandler serves the auth endpoints.
type AuthHandler struct {
	auth     *service.AuthService
	sessions session.Store
	cfg      config.Session
}

// NewAuthHandler builds the auth handler.
func NewAuthHandler(auth *service.AuthService, sessions session.Store, cfg config.Session) *AuthHandler {
	return &AuthHandler{auth: auth, sessions: sessions, cfg: cfg}
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "could not create account")
		return
	}
	response.JSON(c, http.StatusCreated, dto.NewUserResponse(user))
}

// Login handles POST /auth/login: verify credentials, create a session, set
// the cookie.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	user, err := h.auth.Authenticate(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "could not sign in")
		return
	}

	sid, err := h.sessions.Create(c.Request.Context(), session.Session{
		UserID: user.ID,
		Role:   user.Role,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "could not create session")
		return
	}

	middleware.SetSessionCookie(c, h.cfg, sid, int(h.cfg.TTL.Seconds()))
	response.JSON(c, http.StatusOK, dto.NewUserResponse(user))
}

// Logout handles POST /auth/logout: destroy the session and clear the cookie.
// Requires the Auth middleware to have run.
func (h *AuthHandler) Logout(c *gin.Context) {
	if sid, ok := middleware.CurrentSessionID(c); ok {
		_ = h.sessions.Delete(c.Request.Context(), sid)
	}
	middleware.ClearSessionCookie(c, h.cfg)
	response.JSON(c, http.StatusOK, gin.H{"message": "logged out"})
}

// Me handles GET /auth/me: return the current user. Requires Auth middleware.
func (h *AuthHandler) Me(c *gin.Context) {
	id, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.auth.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, "user not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "could not load user")
		return
	}
	response.JSON(c, http.StatusOK, dto.NewUserResponse(user))
}
