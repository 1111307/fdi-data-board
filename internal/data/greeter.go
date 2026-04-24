package data

import (
	"context"
	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/data/orm"
	"gorm.io/gorm"
)

var _ biz.GreeterRepo = (*greeterRepo)(nil)

type greeterRepo struct {
	*baseRepo
}

// NewGreeterRepo .
func NewGreeterRepo(data *Data) biz.GreeterRepo {
	return &greeterRepo{
		baseRepo: &baseRepo{
			data: data,
		},
	}
}

func (r *greeterRepo) db(ctx context.Context) *gorm.DB {
	return r.mysqlDB(ctx)
}

func (r *greeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	gdo := &orm.GreeterDo{
		User: g.User,
	}
	err := r.db(ctx).Create(&gdo).Error
	return g, err
}

func (r *greeterRepo) Update(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	return g, nil
}

func (r *greeterRepo) FindByID(context.Context, int64) (*biz.Greeter, error) {
	return nil, nil
}

func (r *greeterRepo) ListByHello(context.Context, string) ([]*biz.Greeter, error) {
	return nil, nil
}

func (r *greeterRepo) ListAll(context.Context) ([]*biz.Greeter, error) {
	return nil, nil
}
