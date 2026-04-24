package data

import (
	"context"

	"gorm.io/gorm"

	"fdi_data_board/internal/data/orm"
)

type baseRepo struct {
	data *Data
}

func (r *baseRepo) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.mysqlDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = orm.NewTxContext(ctx, tx)
		return fn(ctx)
	})
}

func (r *baseRepo) mysqlDB(ctx context.Context) *gorm.DB {
	tx := orm.TxFromContext(ctx)
	if tx != nil {
		return tx
	}
	return r.data.mysqlDB
}
