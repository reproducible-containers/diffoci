package backend

import (
	"context"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/transfer"
)

type Backend interface {
	Info() Info
	Context(context.Context) context.Context
	ImageService() images.Store
	ContentStore() content.Store
	transfer.Transferrer
	MaybeGC(ctx context.Context) error
}

type Info struct {
	Name string `json:"Name"`
}
