FROM golang:1.24.4-bookworm@sha256:ee7ff13d239350cc9b962c1bf371a60f3c32ee00eaaf0d0f0489713a87e51a67 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
