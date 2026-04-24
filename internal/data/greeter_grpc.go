package data

import (
	"context"

	v1 "fdi_data_board/idl/helloworld/v1"
)

type GreeterGrpcRepo struct {
	data *Data
}

// NewGreeterGrpcRepo .
func NewGreeterGrpcRepo(data *Data) *GreeterGrpcRepo {
	return &GreeterGrpcRepo{
		data: data,
	}
}

func (r *GreeterGrpcRepo) SayHello(ctx context.Context) (*v1.HelloReply, error) {
	client := v1.NewGreeterClient(r.data.anyConn)
	return client.SayHello(ctx, &v1.HelloRequest{
		Name: "hello",
	})
}
