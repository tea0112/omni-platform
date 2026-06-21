package shared

import "context"

type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
