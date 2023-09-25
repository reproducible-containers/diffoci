FROM golang:1.21.1-bookworm@sha256:9c7ea4a4924ae24913401af45c9b6f439d0f782267584738d1cc1099d8b7a02c AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
