ARG GO_BUILD_IMAGE=golang:1.24.13-alpine3.23
ARG SERVER_RUNTIME_IMAGE=alpine:3.23

FROM ${GO_BUILD_IMAGE} AS builder

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/server .

FROM ${SERVER_RUNTIME_IMAGE}

ENV TZ=Asia/Shanghai \
    GVA_CONFIG=/app/config/config.yaml

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 gva \
    && adduser -S -D -H -u 10001 -G gva gva

WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=builder /src/resource /app/resource
COPY config.compose.yaml /app/config.default.yaml
COPY docker-entrypoint.sh /usr/local/bin/gva-entrypoint

RUN chmod 0755 /usr/local/bin/gva-entrypoint \
    && mkdir -p /app/config /app/log /app/uploads \
    && chown -R gva:gva /app

USER gva
EXPOSE 8888

HEALTHCHECK --interval=5s --timeout=3s --start-period=20s --retries=20 \
    CMD wget -qO- http://127.0.0.1:8888/health | grep -q ok || exit 1

ENTRYPOINT ["/usr/local/bin/gva-entrypoint"]
