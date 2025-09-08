FROM golang:1.25.1-bookworm@sha256:6ad9415c04f1cdb7888141cb247cabb28fa74fcc597034494c65d5bc783246a0 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
