package containerdbackend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/log"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend"
	"github.com/reproducible-containers/diffoci/pkg/envutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sys/unix"
)

const Name = "containerd"

func AddFlags(flags *pflag.FlagSet) {
	flags.String("containerd-address", envutil.String("CONTAINERD_ADDRESS", defaultContainerdAddress()),
		"containerd address [$CONTAINERD_ADDRESS]")
	flags.String("containerd-namespace", envutil.String("CONTAINERD_NAMESPACE", namespaces.Default),
		"containerd namespace [$CONTAINERD_NAMESPACE]")
}

func defaultContainerdAddress() string {
	if runtime.GOOS == "linux" && os.Geteuid() != 0 {
		addr, err := rootlessContainerdAddress()
		if err != nil {
			log.L.WithError(err).Debug("Failed to get the address of the rootless containerd (not running?)")
		} else if addr != "" {
			return addr
		}
	}
	return defaults.DefaultAddress
}

func rootlessContainerdAddress() (string, error) {
	xdr := os.Getenv("XDG_RUNTIME_DIR")
	if xdr == "" {
		xdr = fmt.Sprintf("/run/user/%d", os.Geteuid())
	}
	childPidPath := filepath.Join(xdr, "containerd-rootless/child_pid")
	childPidB, err := os.ReadFile(childPidPath)
	if err != nil {
		return "", err
	}
	childPid, err := strconv.Atoi(strings.TrimSpace(string(childPidB)))
	if err != nil {
		return "", fmt.Errorf("failed to parse the content of %q (%q): %w", childPidPath, string(childPidB), err)
	}
	childRoot := fmt.Sprintf("/proc/%d/root", childPid)
	return filepath.Join(childRoot, defaults.DefaultAddress), nil
}

func New(cmd *cobra.Command) (backend.Backend, error) {
	flags := cmd.Flags()
	addr, err := flags.GetString("containerd-address")
	if err != nil {
		return nil, err
	}
	ns, err := flags.GetString("containerd-namespace")
	if err != nil {
		return nil, err
	}
	return newBackend(cmd.Context(), addr, ns)
}

func newBackend(ctx context.Context, addr, ns string) (backend.Backend, error) {
	if err := unix.Access(addr, unix.R_OK); err != nil {
		return nil, fmt.Errorf("failed to access containerd socket %q: %w", addr, err)
	}
	opts := []containerd.ClientOpt{containerd.WithDefaultNamespace(ns)}
	client, err := containerd.New(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd client: %w", err)
	}
	const (
		pluginType = "io.containerd.grpc.v1"
		pluginID   = "transfer"
	)
	plugins, err := client.IntrospectionService().Plugins(ctx, []string{fmt.Sprintf("type==%q,id==%q", pluginType, pluginID)})
	if err != nil {
		return nil, fmt.Errorf("failed to introspect containerd plugins: %w", err)
	}
	if len(plugins.Plugins) == 0 {
		return nil, fmt.Errorf("containerd plugin \"%s.%s\" seems missing (Hint: upgrade containerd to v1.7 or later)", pluginType, pluginID)
	}
	return &containerdBackend{Client: client}, nil
}

type containerdBackend struct {
	*containerd.Client
}

func (b *containerdBackend) Info() backend.Info {
	return backend.Info{
		Name: Name,
	}
}

func (b *containerdBackend) Context(ctx context.Context) context.Context {
	return ctx
}

func (b *containerdBackend) MaybeGC(ctx context.Context) error {
	// NOP
	return nil
}
