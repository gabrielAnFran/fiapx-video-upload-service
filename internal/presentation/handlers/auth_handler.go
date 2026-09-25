package handlers

import (
	"errors"
	"net/http"

	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/application/usecases"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/domain/repositories"
	"github.com/gabrielAnFran/fiapx-video-upload-service/internal/presentation/dto"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	registerUser *usecases.RegisterUserUseCase
	loginUser    *usecases.LoginUserUseCase
}

func NewAuthHandler(registerUser *usecases.RegisterUserUseCase, loginUser *usecases.LoginUserUseCase) *AuthHandler {
	return &AuthHandler{registerUser: registerUser, loginUser: loginUser}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		problem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	user, err := h.registerUser.RegisterUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateEmail) {
			problem(c, http.StatusConflict, "Conflict", "email already registered")
			return
		}
		if errors.Is(err, usecases.ErrEmailRequired) || errors.Is(err, usecases.ErrPasswordRequired) {
			problem(c, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		problem(c, http.StatusInternalServerError, "Internal Server Error", "failed to register user")
		return
	}

	c.JSON(http.StatusCreated, dto.RegisterResponse{UserID: user.ID.String(), Email: user.Email})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		problem(c, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}

	token, err := h.loginUser.LoginUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			problem(c, http.StatusUnauthorized, "Unauthorized", "invalid email or password")
			return
		}
		problem(c, http.StatusInternalServerError, "Internal Server Error", "failed to login")
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token})
}

func problem(c *gin.Context, status int, title, detail string) {
	c.AbortWithStatusJSON(status, dto.NewProblem(status, title, detail, c.Request.URL.Path))
}
