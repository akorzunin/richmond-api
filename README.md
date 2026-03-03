# richmond-api

Backend for richmond app from <https://github.com/Lokrand/richmond>

## Run

```bash
go run main.go
```

build binary

```bash
go build -o ./build/richmond-api
```

### Run DB and s3 in docker

```bash
AUTH_USER=admin AUTH_PASS=admin docker compose -f ./deploy/compose.yaml up -d rustfs postgres
```

## OpenAPI Docs

Install Dependencies

```bash
go mod download
go install github.com/swaggo/swag/v2/cmd/swag@latest
go install github.com/air-verse/air@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
```

After adding/modifying endpoints, regenerate the OpenAPI spec:

```bash
swag init --parseDependency --parseInternal
```

This generates:

- `docs/swagger.json` - OpenAPI 3.1 spec
- `docs/docs.go` - Go bindings (don't edit manually)

Docs at:
<http://localhost:8080/swagger/index.html>

## Migrations ans sqlc

Generate sqlc

```bash
sqlc generate
```

Run migrations locally

```bash
GOOSE_DRIVER=postgres GOOSE_DBSTRING=postgres://admin:admin@localhost:9903/main GOOSE_MIGRATION_DIR=./internal/db/schema goose up
```

Run migrations on host

```bash
GOOSE_DRIVER=postgres GOOSE_DBSTRING=postgres://${AUTH_USER}:${AUTH_PASS}@localhost:9903/main GOOSE_MIGRATION_DIR=./internal/db/schema ~/go/bin/goose up
```

## License

MIT
