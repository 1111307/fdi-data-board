package orm

import (
	"context"

	"gorm.io/gorm"
)

type Transaction interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type contextTxKey struct{}

// TxFromContext returns a Tx stored inside a context, or nil if there isn't one.
func TxFromContext(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(contextTxKey{}).(*gorm.DB)
	return tx
}

// NewTxContext returns a new context with the given Tx attached.
func NewTxContext(parent context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(parent, contextTxKey{}, tx)
}
