# ----- build -------------------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TIME
ARG GIT_COMMIT
ARG GIT_BRANCH

# build dependencies
RUN apk add --no-cache git

WORKDIR /src

# cache dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# build with cache mounts
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w \
    -X 'nagomi-email-parser/internal/version.BuildTime=${BUILD_TIME:-$(date -u +%Y%m%d-%H%M%S)}' \
    -X 'nagomi-email-parser/internal/version.GitCommit=${GIT_COMMIT:-dev}' \
    -X 'nagomi-email-parser/internal/version.GitBranch=${GIT_BRANCH:-main}'" \
    -o /out/nagomi-email-parser ./cmd/server/main.go

# ----- grpc_health_probe -------------------------------------------------------------------------
FROM alpine:3.22.1 AS health-probe

ARG TARGETARCH

RUN apk add --no-cache curl && \
    GRPC_HEALTH_PROBE_VERSION=v0.4.40 && \
    ARCH=${TARGETARCH:-amd64} && \
    curl -fsSL "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-${ARCH}" \
    -o /grpc_health_probe && \
    chmod +x /grpc_health_probe

# ----- runtime -----------------------------------------------------------------------------------
FROM alpine:3.22.1

# metadata
LABEL org.opencontainers.image.title="nagomi-email-parser" \
      org.opencontainers.image.description="SMTP to gRPC email parser" \
      org.opencontainers.image.source="https://github.com/xhos/nagomi-email-parser"

# runtime dependencies
RUN apk add --no-cache ca-certificates tzdata openssl && \
    addgroup -g 1001 -S app && \
    adduser -u 1001 -S -G app -h /app app && \
    mkdir -p /certs && \
    chown -R app:app /certs /app

# copy binaries
COPY --from=builder --chown=app:app /out/nagomi-email-parser /app/nagomi-email-parser
COPY --from=health-probe --chown=app:app /grpc_health_probe /usr/local/bin/grpc_health_probe

# environment defaults
# ENV TZ=UTC \
#     SMTP_ADDR=:25 \
#     EMAIL_PARSER_GRPC_PORT=50053 \
#     SMTP_DOMAIN=localhost \
#     LOG_LEVEL=info

VOLUME ["/certs"]

USER app
WORKDIR /app

EXPOSE 25 50053

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/usr/local/bin/grpc_health_probe", "-addr=:50053"]

ENTRYPOINT ["/app/nagomi-email-parser"]
