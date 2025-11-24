package service

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage/postgres"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type Auth interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(tokenString string) (*models.JWTClaims, error)
}
type AuthService struct {
	userRepo      postgres.UserRepository
	walletRepo    postgres.WalletRepository
	txManager     TxManager
	jwtSecret     []byte
	jwtExpiration time.Duration
	log           *slog.Logger
}

func NewAuthService(
	userRepo postgres.UserRepository,
	walletRepo postgres.WalletRepository,
	txManager TxManager,
	jwtSecret string,
	jwtExpiration time.Duration,
	log *slog.Logger,
) Auth {
	return &AuthService{
		userRepo:      userRepo,
		walletRepo:    walletRepo,
		txManager:     txManager,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiration: jwtExpiration,
		log:           log,
	}
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error) {
	const op = "service.Register"

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", custom_err.ErrInvalidInput, err.Error())
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: failed to hash password: %w", op, err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		createdUser, err := s.userRepo.CreateTx(ctx, tx, user)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		currencies := models.SupportedCurrencies()
		for _, currency := range currencies {
			wallet := &models.Wallet{
				ID:       uuid.New(),
				UserID:   createdUser.ID,
				Currency: string(currency),
				Balance:  0,
			}
			if err := s.walletRepo.CreateWalletTx(ctx, tx, wallet); err != nil {
				return fmt.Errorf("failed to create %s wallet: %w", currency, err)
			}
		}

		s.log.Info("user registered successfully",
			slog.String("op", op),
			slog.String("user_id", createdUser.ID.String()),
			slog.String("username", createdUser.Username))

		return nil
	})

	if err != nil {
		s.log.Error("failed to register user", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.RegisterResponse{
		Message: "User registered successfully",
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	const op = "service.Login"
	const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	user, err := s.userRepo.GetByUsername(ctx, req.Username)

	if err != nil && !errors.Is(err, custom_err.ErrNotFound) {
		s.log.Error("failed to get user", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var hashToCompare string
	if err != nil {
		hashToCompare = dummyHash
	} else {
		hashToCompare = user.PasswordHash
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashToCompare), []byte(req.Password))

	if user == nil || err != nil {
		return nil, custom_err.ErrInvalidCredentials
	}

	token, err := s.generateJWT(user)
	if err != nil {
		s.log.Error("failed to generate JWT", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("user logged in successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("username", user.Username))

	return &models.LoginResponse{
		Token: token,
	}, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	//const op = "service.ValidateToken"

	claims := &models.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {

		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, custom_err.ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, custom_err.ErrTokenNotActive
		}

		return nil, custom_err.ErrInvalidToken
	}

	if !token.Valid {
		return nil, custom_err.ErrInvalidToken
	}

	if claims.UserID == uuid.Nil || claims.Username == "" {
		return nil, custom_err.ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	claims := models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
