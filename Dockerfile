FROM golang:1.23.1-bookworm@sha256:dba79eb312528369dea87532a65dbe9d4efb26439a0feacc9e7ac9b0f1c7f607 AS build-artifacts

RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go \
  --mount=type=bind,src=.,target=/src,rw=true \
  cd /src && \
  make artifacts && \
  cp -a _artifacts /

FROM scratch AS artifacts
COPY --from=build-artifacts /_artifacts/ /
