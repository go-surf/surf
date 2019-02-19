package sqldb

import (
	"context"
	"database/sql"

	"github.com/go-surf/surf"
	"github.com/go-surf/surf/errors"
)

type Database interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) Row

	BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
	Close() error
}

type Transaction interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) Row

	Commit() error
	Rollback() error
}

type Row interface {
	Scan(...interface{}) error
}

var (
	// ErrNotFound is returned when an entity cannot be found.
	ErrNotFound = errors.Wrap(surf.ErrNotFound, "sql")

	// ErrConstraint is returned when an operation cannot be completed due
	// to declared constrained.
	ErrConstraint = errors.Wrap(surf.ErrConstraint, "sql")
)
