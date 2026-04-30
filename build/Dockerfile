# syntax=docker/dockerfile:1

## Build
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

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

RUN apk add --no-cache ca-certificates curl jq

COPY --from=builder /out/omnistrate-ctl /usr/local/bin/omnistrate-ctl
WORKDIR /omnistrate
ENTRYPOINT [ "/usr/local/bin/omnistrate-ctl" ]
