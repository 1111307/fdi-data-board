package data

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fdi_data_board/internal/data/orm"
)

var errDorisNotConfigured = errors.New("doris database not configured")

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
	return r.data.mysqlDB.WithContext(ctx)
}

func (r *baseRepo) dorisDB(ctx context.Context) *gorm.DB {
	if r.data.dorisDB == nil {
		return &gorm.DB{Error: errDorisNotConfigured}
	}
	return r.data.dorisDB.WithContext(ctx)
}
