FROM golang:1.21.6-bookworm@sha256:c4b696f1b2bf7d42e02b62b160c3f81c39386e1c567603df8c514ad6ce93361d AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
