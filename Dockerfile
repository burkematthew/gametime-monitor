FROM golang:1.26 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o gametime-monitor ./cmd/gametime-monitor

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup
COPY --from=builder /app/gametime-monitor /usr/local/bin/gametime-monitor
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
USER appuser
ENTRYPOINT ["entrypoint.sh"]
