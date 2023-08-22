package localbackend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd/content"
	contentlocal "github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/pkg/transfer"
	transferlocal "github.com/containerd/containerd/pkg/transfer/local"
	"github.com/opencontainers/go-digest"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend"
	"github.com/reproducible-containers/diffoci/pkg/envutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.etcd.io/bbolt"
)

const Name = "local"

func AddFlags(flags *pflag.FlagSet) {
	flags.String("local-cache", envutil.String("DIFFOCI_LOCAL_CACHE", defaultLocalCache()),
		"local cache [$DIFFOCI_LOCAL_CACHE]")
}

func defaultLocalCache() string {
	if os.Geteuid() != 0 {
		ucd, err := os.UserCacheDir()
		if err != nil || ucd == "" {
			log.L.WithError(err).Warn("failed to get user cache dir")
		} else {
			return filepath.Join(ucd, "diffoci")
		}
	}
	return "/var/cache/diffoci"
}

func New(cmd *cobra.Command) (backend.Backend, error) {
	flags := cmd.Flags()
	dir, err := flags.GetString("local-cache")
	if err != nil {
		return nil, err
	}
	labelsDir := filepath.Join(dir, "labels")
	for _, f := range []string{dir, labelsDir} {
		if err := os.MkdirAll(f, 0700); err != nil {
			return nil, err
		}
	}
	b := &localBackend{
		ns: "diffoci",
	}
	labelStore := &labelStore{
		dir: labelsDir,
	}
	b.contentStore, err = contentlocal.NewLabeledStore(dir, labelStore)
	if err != nil {
		return nil, err
	}
	dbRaw, err := bbolt.Open(filepath.Join(dir, "diffoci.db"), 0644, nil)
	if err != nil {
		return nil, err
	}
	b.db = metadata.NewDB(dbRaw, b.contentStore, nil)
	b.imageStore = metadata.NewImageStore(b.db)
	lm := metadata.NewLeaseManager(b.db)
	b.transferrer = transferlocal.NewTransferService(lm,
		b.contentStore,
		b.imageStore,
		&transferlocal.TransferConfig{},
	)
	return b, nil
}

type localBackend struct {
	ns           string
	db           *metadata.DB
	contentStore content.Store
	imageStore   images.Store
	transferrer  transfer.Transferrer
}

func (b *localBackend) Info() backend.Info {
	return backend.Info{
		Name: Name,
	}
}

func (b *localBackend) Context(ctx context.Context) context.Context {
	return namespaces.WithNamespace(ctx, b.ns)
}

func (b *localBackend) ContentStore() content.Store {
	return b.contentStore
}

func (b *localBackend) ImageService() images.Store {
	return b.imageStore
}

func (b *localBackend) Transfer(ctx context.Context, source interface{}, destination interface{}, opts ...transfer.Opt) error {
	return b.transferrer.Transfer(ctx, source, destination, opts...)
}

func (b *localBackend) MaybeGC(ctx context.Context) error {
	_, err := b.db.GarbageCollect(ctx)
	return err
}

type labelStore struct {
	dir string
	mu  sync.RWMutex
}

func (ls *labelStore) filepath(d digest.Digest) string {
	return filepath.Join(ls.dir, filepath.Clean(d.Algorithm().String()), filepath.Clean(d.Encoded()))
}

// TODO: flock
func (ls *labelStore) Get(d digest.Digest) (map[string]string, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.getUnlocked(d)
}

func (ls *labelStore) getUnlocked(d digest.Digest) (map[string]string, error) {
	f := ls.filepath(d)
	b, err := os.ReadFile(f)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// TODO: flock
func (ls *labelStore) Set(d digest.Digest, m map[string]string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.setUnlocked(d, m)
}

func (ls *labelStore) setUnlocked(d digest.Digest, m map[string]string) error {
	f := ls.filepath(d)
	if len(m) == 0 {
		return os.RemoveAll(f)
	}
	dir := filepath.Dir(f)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(f, b, 0600)
}

// TODO: flock
func (ls *labelStore) Update(d digest.Digest, m map[string]string) (map[string]string, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	mm, err := ls.getUnlocked(d)
	if err != nil {
		return nil, err
	}
	if mm == nil {
		mm = make(map[string]string)
	}
	for k, v := range m {
		if k == "" {
			delete(mm, k)
		} else {
			mm[k] = v
		}
	}
	return mm, ls.setUnlocked(d, mm)
}
