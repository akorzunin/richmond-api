FROM golang:1.26 AS builder
WORKDIR /build
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o app ./cmd

FROM scratch
WORKDIR /app
COPY --from=builder /build/app .
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./app"]
