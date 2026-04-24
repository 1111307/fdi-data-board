package biz

import (
	"context"

	"fdi_data_board/internal/conf"
)

type BackendServerGroup struct {
	config *conf.Data
}

func NewBackendServerGroup(c *conf.Data) *BackendServerGroup {
	return &BackendServerGroup{config: c}
}

func (bg *BackendServerGroup) Start(_ context.Context) error {
	return nil
}

func (bg *BackendServerGroup) Stop(_ context.Context) error {
	return nil
}
