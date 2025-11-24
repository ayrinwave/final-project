package middlew

import (
	"context"
	"errors"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/service"
	"gw-currency-wallet/pkg/response"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func RequireAuth(authService service.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := GetLogger(r.Context())

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Authorization header is required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				log.Warn("invalid authorization header format")
				response.WriteJSONError(w, log, http.StatusUnauthorized, "unauthorized", "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				switch {
				case errors.Is(err, custom_err.ErrTokenExpired):
					response.WriteJSONError(w, log, http.StatusUnauthorized, "token_expired", "Token has expired")
				case errors.Is(err, custom_err.ErrTokenNotActive):
					response.WriteJSONError(w, log, http.StatusUnauthorized, "token_not_active", "Token not yet active")
				case errors.Is(err, custom_err.ErrInvalidToken):
					response.WriteJSONError(w, log, http.StatusUnauthorized, "invalid_token", "Invalid token")
				default:
					log.Error("failed to validate token", slog.String("error", err.Error()))
					response.WriteJSONError(w, log, http.StatusInternalServerError, "internal_error", "Internal error")
				}
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, userIDKey, claims.UserID)

			loggerWithUser := log.With(slog.String("user_id", claims.UserID.String()))
			ctx = context.WithValue(ctx, loggerKey, loggerWithUser)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) uuid.UUID {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		panic("userID not found in context - RequireAuth middleware not applied?")
	}
	return userID
}
