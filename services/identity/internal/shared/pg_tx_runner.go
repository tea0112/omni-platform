package shared

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxTxRunner struct {
	pool *pgxpool.Pool
}

func NewPgxTxRunner(pool *pgxpool.Pool) TxRunner {
	return &pgxTxRunner{pool: pool}
}

func (r *pgxTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(WithQuerier(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Compile-time check that pgx.Tx satisfies Querier.
var _ Querier = (pgx.Tx)(nil)
