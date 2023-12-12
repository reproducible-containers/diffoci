FROM golang:1.21.5-bookworm@sha256:2d3b13c2a6368032e9697e64e7c923184e6e3be03cf01eadff27de124114e64e AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
