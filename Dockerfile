# Build stage
FROM golang:1.26 AS builder
WORKDIR /build
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/app .
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup
USER appuser
EXPOSE 8080
CMD ["./app"]
