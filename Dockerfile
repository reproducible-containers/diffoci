FROM golang:1.22.4-bookworm@sha256:aec47843e52fee4436bdd3ce931417fa980e9055658b5142140925eea3044bea AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
