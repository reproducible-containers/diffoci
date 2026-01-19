FROM golang:1.26rc2-bookworm@sha256:ab1fe1e98d353ce4f46b60726cdb4aebe3981ec35c6c342f8a219f8a1aced0c3 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
