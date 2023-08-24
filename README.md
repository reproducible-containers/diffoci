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


- - -
# Examples
## Non-reproducible Docker Hub images

### `golang:1.21-alpine3.18`
The sources of the official Docker Hub images are available at <https://github.com/docker-library>.

For example, the source of [`golang:1.21-alpine3.18`](https://hub.docker.com/layers/library/golang/1.21-alpine3.18/images/sha256-dd8888bb7f1b0b05e1e625aa29483f50f38a9b64073a4db00b04076cec52b71c?context=explore)
can be found at <https://github.com/docker-library/golang/blob/d1ff31b86b23fe721dc65806cd2bd79a4c71b039/1.21/alpine3.18/Dockerfile>.

The source can be built as follows:

```console
$ DOCKER_BUILDKIT=0 docker build -t my-golang-1.21-alpine3.18 'https://github.com/docker-library/golang.git#d1ff31b86b23fe721dc65806cd2bd79a4c71b039:1.21/alpine3.18'
...
Successfully tagged my-golang-1.21-alpine3.18:latest
```

> **Note**
>
> `DOCKER_BUILDKIT=0` is specified here because the official `golang:1.21-alpine3.18` image is currently built with the legacy builder.
> A future revision of the official image may be built with BuildKit, and in such a case, `DOCKER_BUILDKIT=1` will rather need to be specified here.

The resulting image binary (`my-golang-1.21-alpine3.18`) can be compared with the official image binary (`golang:1.21-alpine3.18`) as follows:

```console
$ diffoci diff docker://golang:1.21-alpine3.18 docker://my-golang-1.21-alpine3.18 --semantic --report-dir=~/diff
INFO[0000] Loading image "docker.io/library/golang:1.21-alpine3.18" from "docker"
docker.io/library/golang:1.21 alpine3.18        saved
Importing       elapsed: 2.6 s  total:   0.0 B  (0.0 B/s)
INFO[0004] Loading image "docker.io/library/my-golang-1.21-alpine3.18:latest" from "docker"
docker.io/library/my golang 1.21 alpine3        saved
Importing       elapsed: 2.6 s  total:   0.0 B  (0.0 B/s)
TYPE     NAME                      INPUT-0                                                                        INPUT-1
Layer    ctx:/layers-1/layer       length mismatch (457 vs 454)                                                   
File     lib/apk/db/scripts.tar    eef110e559acb7aa00ea23ee7b8bddb52c4526cd394749261aa244ef9c6024a4               342eaa013375398497bfc21dff7dd017a647032ec5c486011142c576b7ccc989
Layer    ctx:/layers-1/layer       name "usr/local/share/ca-certificates/.wh..wh..opq" only appears in input 0    
Layer    ctx:/layers-1/layer       name "usr/share/ca-certificates/.wh..wh..opq" only appears in input 0          
Layer    ctx:/layers-1/layer       name "etc/ca-certificates/.wh..wh..opq" only appears in input 0                
Layer    ctx:/layers-2/layer       length mismatch (13927 vs 13926)                                               
Layer    ctx:/layers-2/layer       name "usr/local/go/.wh..wh..opq" only appears in input 0                       
File     lib/apk/db/scripts.tar    073bb5094fc5bba800f06661dc7f1325c5cb4250b13209fb9e3eaf4e60e4bfc4               1369581b62bd60304c59556ea85f585bd498040c8fa223243622bb7990833063
Layer    ctx:/layers-3/layer       length mismatch (4 vs 3)                                                       
Layer    ctx:/layers-3/layer       name "go/.wh..wh..opq" only appears in input 0  
```

> **Note**
> The `--semantic` flag is specified to ignore differences of timestamps, image names, and other "boring" attributes.
> Without this flag, the `diffoci` command may print an enourmous amount of output.

In the `my-golang-1.21-alpine3.18` image, special files called ["Opaque whiteouts"](https://github.com/opencontainers/image-spec/blob/v1.0.2/layer.md#whiteouts) (`.wh..wh..opq`)
are missing due to filesystem difference between Docker Hub's build machine and the local machine.

Also, the `lib/apk/db/scripts.tar` file in the layer 1 is not reproducible due to the timestamps of the tar entries inside it.
The differences can be inspected by running the [`diffoscope`](https://diffoscope.org/) command for `~/diff/input-{0,1}/layers-1/lib/apk/db/scripts.tar`:
```console
$ sudo apt-get install -y diffoscope

$ diffoscope ~/diff/input-0/layers-1/lib/apk/db/scripts.tar ~/diff/input-1/layers-1/lib/apk/db/scripts.tar
--- /home/suda/diff/input-0/layers-1/lib/apk/db/scripts.tar
+++ /home/suda/diff/input-1/layers-1/lib/apk/db/scripts.tar
├── file list
│ @@ -1,9 +1,9 @@
│ --rwxr-xr-x   0 root         (0) root         (0)       56 2023-08-09 03:36:47.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-install
│ --rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-09 03:36:47.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-install
│ --rwxr-xr-x   0 root         (0) root         (0)      755 2023-08-09 03:36:47.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-upgrade
│ --rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-09 03:36:47.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-upgrade
│ --rwxr-xr-x   0 root         (0) root         (0)      139 2023-08-09 03:36:47.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-install
│ --rwxr-xr-x   0 root         (0) root         (0)     1239 2023-08-09 03:36:47.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-upgrade
│ --rwxr-xr-x   0 root         (0) root         (0)      546 2023-08-09 03:36:47.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.trigger
│ --rwxr-xr-x   0 root         (0) root         (0)      137 2023-08-09 03:36:47.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.post-deinstall
│ --rwxr-xr-x   0 root         (0) root         (0)       63 2023-08-09 03:36:47.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.trigger
│ +-rwxr-xr-x   0 root         (0) root         (0)       56 2023-08-24 07:50:41.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-install
│ +-rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-24 07:50:41.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-install
│ +-rwxr-xr-x   0 root         (0) root         (0)      755 2023-08-24 07:50:41.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-upgrade
│ +-rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-24 07:50:41.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-upgrade
│ +-rwxr-xr-x   0 root         (0) root         (0)      139 2023-08-24 07:50:41.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-install
│ +-rwxr-xr-x   0 root         (0) root         (0)     1239 2023-08-24 07:50:41.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-upgrade
│ +-rwxr-xr-x   0 root         (0) root         (0)      546 2023-08-24 07:50:41.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.trigger
│ +-rwxr-xr-x   0 root         (0) root         (0)      137 2023-08-24 07:50:41.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.post-deinstall
│ +-rwxr-xr-x   0 root         (0) root         (0)       63 2023-08-24 07:50:41.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.trigger
```

These differences are boring, but not filtered out by the `--semantic` flag of the `diffoci` command, because `diffoci` is not aware of the formats of the files inside the image layers.

The `lib/apk/db/scripts.tar` file in the layer 2 has the same issue:
<details>
<p>

```console
$ diffoscope ~/diff/input-0/layers-2/lib/apk/db/scripts.tar ~/diff/input-1/layers-2/lib/apk/db/scripts.tar
--- /home/suda/diff/input-0/layers-2/lib/apk/db/scripts.tar
+++ /home/suda/diff/input-1/layers-2/lib/apk/db/scripts.tar
├── file list
│ @@ -1,9 +1,9 @@
│ --rwxr-xr-x   0 root         (0) root         (0)       56 2023-08-09 04:41:27.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-install
│ --rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-09 04:41:27.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-install
│ --rwxr-xr-x   0 root         (0) root         (0)      755 2023-08-09 04:41:27.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-upgrade
│ --rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-09 04:41:27.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-upgrade
│ --rwxr-xr-x   0 root         (0) root         (0)      139 2023-08-09 04:41:27.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-install
│ --rwxr-xr-x   0 root         (0) root         (0)     1239 2023-08-09 04:41:27.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-upgrade
│ --rwxr-xr-x   0 root         (0) root         (0)      546 2023-08-09 04:41:27.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.trigger
│ --rwxr-xr-x   0 root         (0) root         (0)      137 2023-08-09 04:41:27.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.post-deinstall
│ --rwxr-xr-x   0 root         (0) root         (0)       63 2023-08-09 04:41:27.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.trigger
│ +-rwxr-xr-x   0 root         (0) root         (0)       56 2023-08-24 07:50:52.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-install
│ +-rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-24 07:50:52.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-install
│ +-rwxr-xr-x   0 root         (0) root         (0)      755 2023-08-24 07:50:52.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.pre-upgrade
│ +-rwxr-xr-x   0 root         (0) root         (0)      983 2023-08-24 07:50:52.000000 alpine-baselayout-3.4.3-r1.Q1zwvKMnYs1b6ZdPTBJ0Z7D5P3jyA=.post-upgrade
│ +-rwxr-xr-x   0 root         (0) root         (0)      139 2023-08-24 07:50:52.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-install
│ +-rwxr-xr-x   0 root         (0) root         (0)     1239 2023-08-24 07:50:52.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.post-upgrade
│ +-rwxr-xr-x   0 root         (0) root         (0)      546 2023-08-24 07:50:52.000000 busybox-1.36.1-r2.Q1gQ/L3UBnSjgkFWEHQaUkUDubqdI=.trigger
│ +-rwxr-xr-x   0 root         (0) root         (0)      137 2023-08-24 07:50:52.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.post-deinstall
│ +-rwxr-xr-x   0 root         (0) root         (0)       63 2023-08-24 07:50:52.000000 ca-certificates-20230506-r0.Q1FG8M+7w+dkjV9Vy0mGFWW2t4+Do=.trigger
```

</p>
</details>

Depending on the time to build the image, more differences may happen, especially when the Alpine packages on the internet are bumped up.

#### Conclusion
This example indicates that although the official `golang:1.21-alpine3.18` image binary is not fully reproducible, its non-reproducibility is practically negligible, and
this image binary can be assured to be certainly built from with the [published source](https://github.com/docker-library/golang/blob/d1ff31b86b23fe721dc65806cd2bd79a4c71b039/1.21/alpine3.18/Dockerfile).
**If the published source is trustable**, this image binary can be trusted too.
