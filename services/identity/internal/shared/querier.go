package shared

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type ctxKey struct{}

var querierKey ctxKey

// WithQuerier attaches a Querier (typically a pgx.Tx) to ctx so that
// repository methods called downstream will use it instead of the
// repository's default pool. Callers MUST pass the returned ctx to every
// repo call that should run inside the transaction. Forgetting to do so
// will cause those calls to run outside the tx and silently bypass it.
func WithQuerier(ctx context.Context, q Querier) context.Context {
	return context.WithValue(ctx, querierKey, q)
}

func QuerierFromContext(ctx context.Context) Querier {
	q, _ := ctx.Value(querierKey).(Querier)
	return q
}
