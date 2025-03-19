FROM golang:1.24.1-bookworm@sha256:677d3daa66e708acd3ffadb429e3c749a1070497208db1e97ae71bec26ab4093 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
