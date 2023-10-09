FROM golang:1.21.2-bookworm@sha256:a44d05d5de3474f8135462903bbf74a0fdb761aec455ff557c467339dc0b729b AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
