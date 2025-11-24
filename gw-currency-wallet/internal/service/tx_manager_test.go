package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgxTxManager_WithTx_Success(t *testing.T) {

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectCommit()

	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {

		return nil
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_FunctionError_Rollback(t *testing.T) {

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	expectedErr := errors.New("business logic error")

	mock.ExpectBegin()

	mock.ExpectRollback()

	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		return expectedErr
	})

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_BeginError(t *testing.T) {

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	expectedErr := errors.New("cannot begin transaction")

	mock.ExpectBegin().WillReturnError(expectedErr)

	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		t.Fatal("function should not be called")
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_CommitError(t *testing.T) {

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx := context.Background()

	expectedErr := errors.New("cannot commit transaction")

	mock.ExpectBegin()

	mock.ExpectCommit().WillReturnError(expectedErr)

	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot commit transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPgxTxManager_WithTx_ContextCanceled(t *testing.T) {

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	txManager := NewPgxTxManager(mock)
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	mock.ExpectBegin().WillReturnError(context.Canceled)

	err = txManager.WithTx(ctx, func(tx pgx.Tx) error {
		t.Fatal("function should not be called")
		return nil
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
