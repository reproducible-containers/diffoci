package dockercred

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/pkg/transfer/registry"
	dockerconfig "github.com/docker/cli/cli/config"
	dockerconfigfile "github.com/docker/cli/cli/config/configfile"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
)

func NewCredentialHelper() (registry.CredentialHelper, error) {
	// Load does not raise an error on ENOENT
	dockerConfigFile, err := dockerconfig.Load("")
	if err != nil {
		return nil, err
	}
	h := &helper{
		dockerConfigFile: dockerConfigFile,
	}
	return h, nil
}

type helper struct {
	dockerConfigFile *dockerconfigfile.ConfigFile
}

func (h *helper) GetCredentials(ctx context.Context, ref, origHost string) (registry.Credentials, error) {
	hosts := []string{origHost}
	if origHost == "registry-1.docker.io" {
		hosts = append(hosts, "https://index.docker.io/v1/")
	} else if !strings.Contains(origHost, "://") {
		hosts = append(hosts, "https://"+origHost, "http://"+origHost)
	}

	var emptyAC dockerconfigtypes.AuthConfig
	for _, host := range hosts {
		ac, err := h.dockerConfigFile.GetAuthConfig(host)
		if err != nil {
			return registry.Credentials{}, fmt.Errorf("failed to call GetAutoConfig(%q): %w", host, err)
		}
		if ac == emptyAC {
			continue
		}
		cred := registry.Credentials{
			Host:     ac.ServerAddress,
			Username: ac.Username,
			Secret:   ac.Password,
		}
		if ac.IdentityToken != "" {
			cred.Username = ""
			cred.Secret = ac.IdentityToken
		}
		return cred, nil
	}
	return registry.Credentials{}, nil
}
