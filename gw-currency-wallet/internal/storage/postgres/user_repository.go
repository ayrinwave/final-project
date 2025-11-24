package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateTx(ctx context.Context, tx pgx.Tx, user *models.User) (*models.User, error)

	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
}
type PgUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &PgUserRepository{db: db}
}

func (r *PgUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	return r.execCreate(ctx, r.db, user)
}

func (r *PgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	const op = "storage.GetByID"

	var user models.User
	err := r.db.QueryRow(ctx, storage.GetUserByIDQuery, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (r *PgUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	const op = "storage.GetByUsername"

	var user models.User
	err := r.db.QueryRow(ctx, storage.GetUserByUsernameQuery, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}
