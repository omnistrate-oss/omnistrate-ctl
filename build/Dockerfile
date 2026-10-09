# syntax=docker/dockerfile:1

## Build
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

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

## Deploy
FROM alpine:latest AS app

# The published alpine:latest image is rebuilt less often than the Alpine package
# repository is updated, so its preinstalled packages can carry already-fixed CVEs — the
# OpenSSL libraries that apk-tools links against are the usual offenders. Upgrading before
# installing pulls those forward to whatever the repository currently ships, which keeps
# the image clean as new advisories land rather than only at the next base-image rebuild.
RUN apk --no-cache upgrade && apk add --no-cache ca-certificates curl jq

# zlib 1.3.1.2 through 1.3.2 carry CVE-2026-85091 (high, heap overflow in gz_vacate), and every
# supported Alpine branch ships 1.3.2-r0 with no fix yet. It cannot be dropped — libcurl and
# apk-tools link against it — so pin the last unaffected release from v3.19, whose ABI is
# compatible. The repository is passed for this one command only and is not persisted.
# Remove this pin once Alpine publishes a fixed zlib, otherwise the image stays on 1.3.1.
RUN apk add --no-cache --repository https://dl-cdn.alpinelinux.org/alpine/v3.19/main 'zlib=1.3.1-r0'

COPY --from=builder /out/omnistrate-ctl /usr/local/bin/omnistrate-ctl
WORKDIR /omnistrate
ENTRYPOINT [ "/usr/local/bin/omnistrate-ctl" ]
