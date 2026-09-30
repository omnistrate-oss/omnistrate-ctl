# syntax=docker/dockerfile:1

## Build
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /go/src/ctl

ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY
ARG GOSUMDB
ARG GIT_COMMIT
ARG GIT_VERSION

ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GIT_COMMIT=${GIT_COMMIT} \
    GIT_VERSION=${GIT_VERSION}

# Download modules first to maximize layer cache reuse across source changes.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

# Bring in the rest of the source via bind mount to avoid an extra COPY layer
# and rely on BuildKit cache mounts for the module and build caches.
RUN --mount=type=bind,source=.,target=/go/src/ctl,rw \
    --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,id=gobuild-${TARGETOS}-${TARGETARCH} \
    BUILD_INFO_PKG=github.com/omnistrate-oss/omnistrate-ctl/internal/config && \
    BUILD_TIMESTAMP=$(date -u +'%Y-%m-%dT%H:%M:%SZ') && \
    go build -mod=mod -trimpath \
      -ldflags "-s -w \
        -X ${BUILD_INFO_PKG}.CommitID=${GIT_COMMIT} \
        -X ${BUILD_INFO_PKG}.Timestamp=${BUILD_TIMESTAMP} \
        -X ${BUILD_INFO_PKG}.Version=${GIT_VERSION}" \
      -o /out/omnistrate-ctl \
      github.com/omnistrate-oss/omnistrate-ctl

## Patched runtime dependency (built for the target architecture)
FROM alpine:3.24 AS zlib-builder
RUN --mount=type=bind,source=build/zlib.sh,target=/build-zlib.sh \
    sh /build-zlib.sh

## Deploy
FROM alpine:3.24 AS app

# The published alpine:latest image is rebuilt less often than the Alpine package
# repository is updated, so its preinstalled packages can carry already-fixed CVEs — the
# OpenSSL libraries that apk-tools links against are the usual offenders. Upgrading before
# installing pulls those forward to whatever the repository currently ships, which keeps
# the image clean as new advisories land rather than only at the next base-image rebuild.
# Only the locally built, checksum-verified package is allowed to be unsigned.
RUN --mount=type=bind,from=zlib-builder,source=/out/zlib.apk,target=/tmp/zlib.apk \
    apk --no-cache upgrade && apk add --no-cache ca-certificates curl jq && \
    apk add --no-cache --no-network --allow-untrusted /tmp/zlib.apk

COPY --from=builder /out/omnistrate-ctl /usr/local/bin/omnistrate-ctl
WORKDIR /omnistrate
ENTRYPOINT [ "/usr/local/bin/omnistrate-ctl" ]
