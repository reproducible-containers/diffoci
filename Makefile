# Files are installed under $(DESTDIR)/$(PREFIX)
PREFIX ?= /usr/local
DEST := $(shell echo "$(DESTDIR)/$(PREFIX)" | sed 's:///*:/:g; s://*$$::')

VERSION ?=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always --tags)
VERSION_SYMBOL := github.com/reproducible-containers/diffoci/cmd/diffoci/version.Version

export CGO_ENABLED ?= 0
export DOCKER_BUILDKIT := 1
export SOURCE_DATE_EPOCH ?= $(shell git log -1 --pretty=%ct)

GO ?= go
GO_LDFLAGS ?= -s -w -X $(VERSION_SYMBOL)=$(VERSION)
GO_BUILD ?= $(GO) build -trimpath -ldflags="$(GO_LDFLAGS)"
DOCKER ?= docker
DOCKER_BUILD ?= $(DOCKER) build --build-arg SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)

.PHONY: all
all: binaries

.PHONY: binaries
binaries: _output/bin/diffoci

.PHONY: _output/bin/diffoci
_output/bin/diffoci:
	$(GO_BUILD) -o $@ ./cmd/diffoci

.PHONY: install
install: uninstall
	mkdir -p "$(DEST)"
	install _output/bin/diffoci "$(DEST)/bin/diffoci"

.PHONY: uninstall
uninstall:
	rm -rf "$(DEST)/bin/diffoci"

.PHONY: clean
clean:
	rm -rf _output _artifacts

.PHONY: artifacts
artifacts:
	rm -rf _artifacts
	mkdir -p _artifacts
	GOOS=linux  GOARCH=amd64            $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).linux-amd64   ./cmd/diffoci
	GOOS=linux  GOARCH=arm      GOARM=7 $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).linux-arm-v7  ./cmd/diffoci
	GOOS=linux  GOARCH=arm64            $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).linux-arm64   ./cmd/diffoci
	GOOS=linux  GOARCH=ppc64le          $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).linux-ppc64le ./cmd/diffoci
	GOOS=linux  GOARCH=riscv64          $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).linux-riscv64 ./cmd/diffoci
	GOOS=linux  GOARCH=s390x            $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).linux-s390x   ./cmd/diffoci
	GOOS=darwin GOARCH=amd64            $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).darwin-amd64  ./cmd/diffoci
	GOOS=darwin GOARCH=arm64            $(GO_BUILD) -o _artifacts/diffoci-$(VERSION).darwin-arm64  ./cmd/diffoci
	(cd _artifacts ; sha256sum *) > SHA256SUMS
	mv SHA256SUMS _artifacts/SHA256SUMS
	touch -d @$(SOURCE_DATE_EPOCH) _artifacts/*

.PHONY: artifacts.docker
artifacts.docker:
	$(DOCKER_BUILD) --output=./_artifacts --target=artifacts .
