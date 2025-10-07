package repositories

import (
	"context"
	"database/sql"
)

type txKey struct{}

type sqlTx interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func injectTx(ctx context.Context, db sqlTx) context.Context {
	return context.WithValue(ctx, txKey{}, db)
}

func (r *Repository) extractTxWrite(ctx context.Context) sqlTx {
	if db, ok := ctx.Value(txKey{}).(sqlTx); ok {
		return db
	}
	return r.dbWrite
}

func (r *Repository) extractTxRead(ctx context.Context) sqlTx {
	if db, ok := ctx.Value(txKey{}).(sqlTx); ok {
		return db
	}
	return r.dbRead
}
