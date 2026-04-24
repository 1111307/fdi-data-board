package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode"
	"devops.momenta.works/Momenta/FDI/_git/gerr.git/gerror"

	v1 "fdi_data_board/idl/helloworld/v1"
	"fdi_data_board/internal/data/orm"
)

var (
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

type Greeter struct {
	User string
}

type GreeterRepo interface {
	orm.Transaction
	Save(context.Context, *Greeter) (*Greeter, error)
	Update(context.Context, *Greeter) (*Greeter, error)
	FindByID(context.Context, int64) (*Greeter, error)
	ListByHello(context.Context, string) ([]*Greeter, error)
	ListAll(context.Context) ([]*Greeter, error)
}

type GreeterUsecase struct {
	repo GreeterRepo
}

func NewGreeterUsecase(repo GreeterRepo) *GreeterUsecase {
	return &GreeterUsecase{repo: repo}
}

func (uc *GreeterUsecase) CreateGreeter(ctx context.Context, g *Greeter) (*Greeter, error) {
	log.Context(ctx).Infof("CreateGreeter: %v", g.User)
	if err := uc.repo.InTx(ctx, func(ctx context.Context) error {
		_, err := uc.repo.Save(ctx, g)
		return err
	}); err != nil {
		return nil, gerror.WrapCode(gcode.CodeInternalError, err)
	}
	return g, nil
}
