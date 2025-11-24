package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Интерфейс для query executor
type pgxQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (r *PgUserRepository) CreateTx(ctx context.Context, tx pgx.Tx, user *models.User) (*models.User, error) {
	return r.execCreate(ctx, tx, user)
}
func (r *PgUserRepository) execCreate(ctx context.Context, q pgxQueryer, user *models.User) (*models.User, error) {
	const op = "storage.execCreate"

	var createdUser models.User
	err := q.QueryRow(
		ctx,
		storage.CreateUserQuery,
		user.ID, user.Username, user.Email, user.PasswordHash,
	).Scan(
		&createdUser.ID,
		&createdUser.Username,
		&createdUser.Email,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "username") {
				return nil, custom_err.ErrUsernameExists
			}
			if strings.Contains(pgErr.ConstraintName, "email") {
				return nil, custom_err.ErrEmailExists
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	createdUser.PasswordHash = user.PasswordHash
	return &createdUser, nil
}
