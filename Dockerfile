FROM golang:1.21.1-bookworm@sha256:61f84bc8cddb878258b2966d682c11a1317e97a371ff0da98823d9e326d9dac1 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
