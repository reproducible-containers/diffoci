package backend

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/pkg/transfer"
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
