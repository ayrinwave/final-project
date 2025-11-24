package handlers

import (
	"encoding/json"
	"errors"
	"gw-currency-wallet/internal/api/middlew"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/pkg/response"
	"log/slog"
	"net/http"
)

type AuthHandler struct {
	service service.Auth
}

func NewAuthHandler(service service.Auth) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// Register godoc
// @Summary      Регистрация пользователя
// @Description  Создает нового пользователя и кошельки для всех валют
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.RegisterRequest true "Данные регистрации"
// @Success      201 {object} models.RegisterResponse
// @Failure      400 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	log := middlew.GetLogger(r.Context())
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.handleRegisterError(w, log, err)
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusCreated, resp)
}

func (h *AuthHandler) handleRegisterError(w http.ResponseWriter, log *slog.Logger, err error) {
	switch {
	case errors.Is(err, custom_err.ErrUsernameExists):
		response.WriteJSONError(w, log, http.StatusBadRequest, "username_exists", "Username already exists")
	case errors.Is(err, custom_err.ErrEmailExists):
		response.WriteJSONError(w, log, http.StatusBadRequest, "email_exists", "Email already exists")
	case errors.Is(err, custom_err.ErrInvalidInput):
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		log.Error("failed to register user", slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "Internal error")
	}
}

// Login godoc
// @Summary      Авторизация пользователя
// @Description  Авторизует пользователя и возвращает JWT токен
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.LoginRequest true "Данные входа"
// @Success      200 {object} models.LoginResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	const op = "handler.Login"
	log := middlew.GetLogger(r.Context())

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid JSON body", slog.String("op", op), slog.String("error", err.Error()))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if req.Username == "" {
		log.Warn("username is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "username is required")
		return
	}
	if req.Password == "" {
		log.Warn("password is required", slog.String("op", op))
		response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_field", "password is required")
		return
	}

	log.Info("user login attempt", slog.String("op", op), slog.String("username", req.Username))

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, custom_err.ErrInvalidCredentials):
			log.Info("invalid credentials", slog.String("op", op), slog.String("username", req.Username))
			response.WriteJSONError(w, log, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
		case errors.Is(err, custom_err.ErrInvalidInput):
			log.Warn("invalid input", slog.String("op", op))
			response.WriteJSONError(w, log, http.StatusBadRequest, "invalid_input", "Invalid input data")
		default:
			log.Error("failed to login user", slog.String("op", op), slog.String("error", err.Error()))
			response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "An internal error occurred")
		}
		return
	}

	response.WriteJSONSuccess(w, log, http.StatusOK, resp)
}
