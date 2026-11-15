package testretag

import "github.com/reproducible-containers/diffoci/golden/types"

func tagged(registry, repo, tag string) types.Image {
	return types.Image{
		Layers: [][]types.Entry{{types.File("app", "#!/bin/sh\n")}},
		Annotations: map[string]string{
			"containerd.io/distribution.source." + registry: repo,
			"io.containerd.image.name":                      registry + "/" + repo + ":" + tag,
			"org.opencontainers.image.ref.name":             tag,
		},
	}
}

var Tests = types.GoldenTests{
	Name: "test_retag",
	Images: [2]types.Image{
		tagged("docker.io", "example/app", "1.0"),
		tagged("registry.example.com", "mirror/app", "latest"),
	},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--ignore-image-name"}, Output: "ignore_image_name"},
	},
}
