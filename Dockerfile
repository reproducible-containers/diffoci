FROM golang:1.23.1-bookworm@sha256:1a5326b07cbab12f4fd7800425f2cf25ff2bd62c404ef41b56cb99669a710a83 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
