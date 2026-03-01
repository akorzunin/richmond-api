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

## License

MIT
