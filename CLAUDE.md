# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**fdi_data_board** is a Kratos-based Go microservice that provides a data dashboard backend with multi-protocol support (gRPC, HTTP, Gin). It manages query scenes, dashboards, data sources, and provides real-time data visualization capabilities integrated with Doris OLAP database, MySQL, and Redis.

**Key Technologies:**
- Go 1.21
- Kratos v2 framework (gRPC microservice framework)
- Gin web framework for REST API
- GORM for ORM
- Protocol Buffers for API definition
- Doris (OLAP database) for analytics data
- MySQL for metadata
- Redis for caching

## Architecture Overview

### Layered Architecture (Clean Architecture)

The codebase follows a clean architecture with wire-based dependency injection:

1. **Transport Layer** (`internal/server/`): Protocol handling
   - `grpc.go` - gRPC server setup (port 9012)
   - `http.go` - HTTP Kratos gateway server (port 8000) with OpenAPI/Swagger
   - `gin.go` - Gin REST API server (port 8081) with swagger-ui
   - `simple.go` - Stateless event server
   - `server.go` - Server initialization using Wire

2. **Service Layer** (`internal/service/`): Business logic orchestration
   - `greeter_api.go`, `greeter_rpc.go` - Example service
   - `query_scene.go` - Query scene management
   - `datasource.go` - Data source configuration
   - `scene_group.go` - Scene grouping
   - `dashboard_fo.go` (FO = Front Office), `dashboard_do.go` (DO = Design Office)
   - `service.go` - Wire provider set

3. **Business Logic Layer** (`internal/biz/`): Use cases
   - `use_query_scene.go` - Scene query business logic with QueryResult DTO
   - `use_datasource.go` - Data source management
   - `use_scene_group.go` - Scene group management
   - `use_dashboard_*.go` - Dashboard use cases
   - `server_group.go` - Server group configuration
   - `biz.go` - Wire provider set

4. **Data Layer** (`internal/data/`): Repository pattern & database access
   - `data.go` - Database initialization and connection pooling
     - MySQL: metadata store (query scenes, data sources, dashboards)
     - Doris: OLAP analytics data (optional, configurable)
     - gRPC client: integration with external services
     - Dynamic datasource cache: supports multiple database connections
   - `*_[datasource|query_scene|scene_group].go` - Repository implementations
   - `orm/` - GORM models and schema definitions

5. **HTTP Routes** (`internal/route/`): Gin route registration
   - `route.go` - Wire provider that aggregates all service routes
   - `ginhandler.go` - Gin handler utilities
   - `greeter.go`, `query_scene.go`, `dashboard.go` - Route groups

6. **Entry Point** (`cmd/server/`)
   - `main.go` - Application bootstrap
   - `wire.go` - Dependency injection definition (generates `wire_gen.go`)

7. **API Definitions** (`api/` and `idl/`)
   - `.proto` files define gRPC and HTTP APIs
   - Kratos generates `.pb.go` and `*_http.pb.go` files automatically

### Data Model

Key ORM models in `internal/data/orm/`:
- `QuerySceneDo` - Query scenarios (scenes) with parameters and widgets
- `QuerySceneParamDo` - Scene parameters (text, number, select, date types)
- `QuerySceneWidgetDo` - Dashboard widgets with SQL queries
- `QuerySceneGroupDo` - Scene grouping/organization
- `QueryDatasourceDo` - Data source configurations
- `SlowQueryLogDo` - Performance monitoring
- `GreeterDo` - Example model

### Configuration

- **Config files:** `configs/config-{dev,prd}.yaml`
- **Bootstrap proto:** `internal/conf/conf.proto` defines:
  - Server: HTTP, gRPC, Gin addresses and timeouts
  - Data: MySQL, Redis, Keycloak, gRPC client, Doris, Runtime settings
- **Environment variables:** Override config values (e.g., `MYSQL_USERNAME`, `RUN_MODE`)

## Build, Test & Development Commands

### Setup & Dependencies

```bash
# Initialize environment (download protoc plugins and tools)
make init

# Download submodule (cloud-idl with proto definitions)
make submodule

# Install Go tools
cd cmd/server && go install github.com/google/wire/cmd/wire@latest
```

### Code Generation

```bash
# Generate all (api + config + swagger + wire)
make all

# Generate API from proto files (pb.go, http, grpc, OpenAPI)
make api

# Generate internal config structures from proto
make config

# Generate Swagger documentation
make swagger

# Generate wire dependency injection
cd cmd/server && wire
```

### Building & Running

```bash
# Build binary
make build

# Run server (requires config-dev.yaml)
go run cmd/server/main.go -conf configs/config-dev.yaml

# Docker build & run
docker build -t fdi-data-board .
docker run --rm -p 8000:8000 -p 9000:9000 -p 8081:8081 -e RUN_MODE=dev fdi-data-board
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./internal/biz/... -v

# Run specific test
go test -run TestQueryScene ./internal/biz/... -v

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
```

### Code Quality

```bash
# Format code
go fmt ./...

# Vet for suspicious code
go vet ./...

# Tidy dependencies
go mod tidy
```

## API Endpoints

### Gin Server (Port 8081)

Primary REST API for frontend consumption. Swagger UI available at `/swagger/index.html`

Key route groups:
- `/query_scene/v1/*` - Query scene operations (list, detail, save, delete, execute)
- `/dashboard/*` - Dashboard operations (FO and DO variants)
- `/datasource/v1/*` - Data source management
- `/scene_group/v1/*` - Scene grouping operations
- `/healthy` - Health check

### HTTP Server (Port 8000)

Kratos HTTP gateway converting gRPC to HTTP. OpenAPI at `/q/swagger-ui`

### gRPC Server (Port 9012)

gRPC endpoint for microservice-to-microservice communication

## Key Development Patterns

### Dependency Injection with Wire

The `wire.go` files define dependency providers:
- Providers use the `wire.NewSet()` pattern
- `wireApp()` in `cmd/server/wire.go` bootstraps all layers
- Run `wire` in `cmd/server/` after adding new providers

### Service Request/Response Pattern

Services return `(api.HttpResponse, error)` where responses implement:
```go
type HttpResponse interface {
    GetCode() int32
    GetMessage() string
}
```

Use proto-generated response types or custom structs with Code and Message fields.

### Database Access

**Cached dynamic datasources:** The `Data` struct maintains sync.Map caches:
- `dsCache`: datasource connections (keyed by datasource_id)
- `dimCache`: dimension values for lookups
- Always call `InvalidateDatasourceCache()` after datasource changes

**Doris integration:** Optional OLAP engine for analytics queries (enabled via `DORIS_ON` environment variable). MySQL used when Doris unavailable.

### Error Handling

Uses custom error package `devops.momenta.works/Momenta/FDI/_git/gerr.git/gcode` for standardized error codes with HTTP status mappings. Return errors with appropriate `gcode.Code` values.

## Important Files & Patterns

- **Configuration bootstrap:** `cmd/server/main.go` → loads `config-{dev|prd}.yaml` → applies env var overrides
- **Router setup:** `internal/route/route.go` aggregates service route handlers
- **GORM auto-migration:** `internal/data/data.go` creates tables on startup
- **Swagger generation:** Uses swag tool with struct comments (see `internal/service/query_scene.go` for examples)
- **Proto-to-HTTP mapping:** Kratos uses `google.api.http` annotations for route mapping

## Deployment

Docker image builds using multi-stage process in `Dockerfile`:
- Builder stage: compiles protobuf, generates code, builds binary
- Runtime stage: minimal Alpine image with binary and config files
- Exposed ports: 8000 (HTTP), 9000 (gRPC - legacy), 8080 (Gin - legacy naming)
- Entry: `./server -conf ./configs/config-${RUN_MODE}.yaml`

Azure Pipelines CI/CD triggers on tags (v*) and branches (master, develop, feature/*, release/*).
