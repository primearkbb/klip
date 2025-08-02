# CRUSH - Klip Development Guide

## Build/Test Commands
- `make build` - Build for current platform
- `make test` - Run all tests
- `go test ./internal/app -v` - Run single package tests
- `go test -run TestNew ./internal/app` - Run specific test
- `make test-coverage` - Run tests with coverage report
- `make lint` - Run golangci-lint (requires installation)
- `make fmt` - Format all Go code
- `make vet` - Run go vet
- `make check` - Run fmt, vet, lint, and test
- `make dev` - Run with live reload (requires air)

## Code Style Guidelines
- **Imports**: Standard library first, then third-party, then internal packages with blank lines between groups
- **Naming**: Use camelCase for variables/functions, PascalCase for exported types, snake_case for JSON tags
- **Types**: Always use explicit types for struct fields, prefer interfaces for dependencies
- **Error Handling**: Always check errors, wrap with context using fmt.Errorf, return early on errors
- **Comments**: Document all exported functions/types, use // for single line, /* */ for blocks
- **Testing**: Use testify/assert, table-driven tests preferred, test files end with _test.go
- **Structs**: Group related fields, put exported fields first, use struct tags for JSON/config
- **Dependencies**: Use dependency injection, prefer interfaces over concrete types
- **Context**: Always pass context.Context as first parameter for functions that may block
- **Logging**: Use charmbracelet/log, structured logging with key-value pairs
- **Constants**: Group related constants in blocks, use iota for enums
- **File Organization**: Keep files focused, separate concerns, use internal/ for private packages