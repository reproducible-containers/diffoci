# diffoci: diff for Docker and OCI container images

`diffoci` compares Docker and OCI container images for helping [reproducible builds](https://reproducible-builds.org/).

> **Note**
> "OCI" here refers to the "[Open Container Initiative](https://opencontainers.org/)", not to the "Oracle Cloud Infrastructure".

```console
$ diffoci diff --semantic alpine:3.18.2 alpine:3.18.3
TYPE    NAME                                 INPUT-0                                                             INPUT-1
File    lib/libcrypto.so.3                   4518b7d5f6563f81ca1b3d857bbd7008d686545d8211283240506449618b2858    d21ccbb32c1e98340027504e4d951ab4c38450f1f211a4157fa9e962216ac625
File    usr/bin/iconv                        8c54b8962ef07047032facf57cacb2f19eecfbefe03ad001e8c6c2b332e0d334    07115db0a4bd082f746ddbc6433b96397adcb523956417cb8376cd1ab4faf3d6
File    usr/lib/engines-3/capi.so            1b0508fa6be2efe1412f59da46b3963c7611696563b6b2b699b1aa9dba447b5a    cc03f5d58b206389f9560e8e7f15495d2bd2f384793549f63f3cb3cb9c71b466
File    bin/busybox                          6bf2a4358d5c26033d2d2e30ef9ce0ba9e3a9b34d81f0964f1d127123ae3e827    89f85a8b92bdb5ada51bd8ee0d9d687da1a759a0d80f8d012e7f8cbb3933a3a0
File    etc/os-release                       fb844374742438cf1b4e675dcd7d87c2fd6fbdb7cc7be30c62d4027240474aaf    e08e943282c5d38f99bfde311c7d5759a4578f92fca5943e5b1351e8cd472892
File    usr/lib/engines-3/padlock.so         670f9a084f97974b12c81cc7a88fce5ec9f47dcee00da3192a17577a5298e62f    15fb67a501ad1293034dd16106edbdee0c07803ebcbbeb5a8123b4e784c1491f
File    usr/bin/getconf                      d314880d6288d48e8223b3da02d00605f6a993b669593c1c1a0fa5fd95339d48    040fc0ddb83193809919381bb23d89e4efefdc207e18ef0f3d0e0d9977bf2b58
File    lib/ld-musl-x86_64.so.1              c6b3288ba48945a21ede7ccf6f7a257d41bacf2c061236726ff9b5def383a766    35002c47957674f588389c0f3ca23b01adab091ea5773326e1c7782e5ece1207
File    usr/lib/engines-3/afalg.so           2aaa6a94f0e07f556e77faa8bec55321d0fe138f27193fdcd6ef8ccef051c7ba    9ef6555b3594798ce5a05dfd79b996397642b367c75b5cab0c752050d28e6138
File    lib/apk/db/installed                 d8b49a733ba6590033ae75f27a52f83281b693ba46751a9e488bf12dbdb06c61    c5cd4d9ed0a78abe602cc1e5fb29c84bef033c99d7c7f5e15fdb76b6c6cf8ad0
File    lib/apk/db/scripts.tar               fbc51430c9f8b6a1a547281858fb36dff7947a2660a8bb6937b48f12361e3bb1    7d28a05788bda8fc5ec835257d5204d6d45c002458b04ad4981bce3be733c80f
File    etc/alpine-release                   8fc1c65f6ef13f8ef573d59100eb183cf5a66aa7e94a6e6cba1dd22e7a0af51b    7cc26f207ef55a1422daecc84d155e9fa50fd170574807e73e55823dd81407d9
File    etc/ssl/misc/tsget.pl                559f94eb47d2d6f81c361e334fd882596858c7934a8b0111177f535a910f2990    3feab60fb8d102da8746aa2a422ba77c60dda6b4ae9ac33056fe82bfc071969d
File    lib/apk/db/triggers                  f78b6968b2c85c453a9dd89368a9fc78788d7c34aeb202544a41714e90544a37    ff37f223b335fc42d5a563dd268cbf476411d8fdd2b88addade5b5f715f0425a
File    lib/libssl.so.3                      c22f9f45c1266ebdcc5f89c2f3471e89ffc5b2a4cd299f672156723270994133    91ad7b1cc8cf5575afeeb202c0fd1fefb63ceea1492491218f3132813eb04e49
File    usr/lib/engines-3/loader_attic.so    4f419a09b9f608e753045d83c255818c793913c274f724eb10d8ca8f474ae457    959301d5c88fab262a0631f1c87d02e1ef52215de35e0316af4513014a449b7f
File    usr/bin/getent                       1c6aa066998ddd019b6df0142ffcf8b1f4307593dc42b92e7fc82cd601905f75    c22523bc4d2208c8df51b9c7b87ad5596aaf171a4e69d169f007ada8241c5d09
File    usr/lib/ossl-modules/legacy.so       5b677eca0c3a3ac53c1a49fbac49534ea2862c62416d14ec6053f2080d5bae50    65b55bc81cbab89e7d2d7e5636455ef86642ae8c377dec3924103d3513ede888
```

## Installation
### Binary
Binaries are available for Linux and macOS: https://github.com/reproducible-containers/diffoci/releases

### Source
Needs Go 1.21 or later.
```bash
go install github.com/reproducible-containers/diffoci/cmd/diffoci@latest
```

## Basic usage
### Strict mode
```bash
diffoci diff IMAGE0 IMAGE1
```

The strict mode is often too strict.
Consider using the non-strict mode (see below).

### Non-strict (aka "semantic") mode
```bash
diffoci diff --semantic IMAGE0 IMAGE1
```

Non-strict mode ignores:
- Timestamps
- Build histories
- File ordering in tar layers
- Image name annotations

## Advanced usage
### Dumping conflicting files
Set `--report-dir=DIR` to dump conflicting files in the specified dir.

```console
$ diffoci diff --semantic --report-dir=/tmp/diff alpine:3.18.2 alpine:3.18.3
TYPE    NAME                                 INPUT-0                                                             INPUT-1
...
File    etc/alpine-release                   8fc1c65f6ef13f8ef573d59100eb183cf5a66aa7e94a6e6cba1dd22e7a0af51b    7cc26f207ef55a1422daecc84d155e9fa50fd170574807e73e55823dd81407d9
...

$ diff -ur /tmp/diff/input-0/ /tmp/diff/input-1/
Binary files /tmp/diff/input-0/manifests-0/layers-0/bin/busybox and /tmp/diff/input-1/manifests-0/layers-0/bin/busybox differ
diff -ur /tmp/diff/input-0/manifests-0/layers-0/etc/alpine-release /tmp/diff/input-1/manifests-0/layers-0/etc/alpine-release
--- /tmp/diff/input-0/manifests-0/layers-0/etc/alpine-release   2023-06-15 00:03:14.000000000 +0900
+++ /tmp/diff/input-1/manifests-0/layers-0/etc/alpine-release   2023-08-07 22:09:12.000000000 +0900
@@ -1 +1 @@
-3.18.2
+3.18.3
...
```

### Accessing containerd images
`diffoci` uses the containerd image store by default when containerd v1.7 or later is running.
The default namespace is `default`.

To explicitly enable containerd:
```bash
diffoci --backend=containerd --containerd-socket=SOCKET --containerd-namespace=NAMESPACE
```

To explicitly disable containerd:
```bash
diffoci --backend=local
```

### Accessing Docker images
To access Docker images that are not pushed to a registry, prepend `docker://` to the image name:
```bash
docker build -t foo ~/foo
docker build -t bar ~/bar
diffoci diff docker://foo docker://bar
```

You do NOT need to specify a custom `--backend` to access Docker images.

### Accessing Podman images
To access Podman images that are not pushed to a registry, prepend `podman://` to the image name.
See the `docker://` example above, and read `docker` as `podman`.

### Accessing private images
To access private images, create a credential file as `~/.docker/config.json` using `docker login`.
