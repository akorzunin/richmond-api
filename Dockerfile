FROM golang:1.26 AS builder
WORKDIR /build
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o app ./cmd

FROM scratch AS runner
WORKDIR /app
COPY --from=builder /build/app .
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./app"]

FROM scratch AS migrations-runner
ARG GOOSE_VERSION=v3.27.0
ADD --chmod=0755 https://github.com/pressly/goose/releases/download/${GOOSE_VERSION}/goose_linux_x86_64 /bin/goose
COPY ./internal/db/schema /migrations
CMD ["/bin/goose", "up"]
