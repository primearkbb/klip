# Klip Go - Terminal AI Chat Application

A modern terminal-based AI chat application built with Go and the Charm ecosystem libraries.

## Architecture

This is the Go rewrite of the original Deno/TypeScript Klip application, using:

- **Bubble Tea** - For terminal user interface framework
- **Bubbles** - Pre-built UI components  
- **Lipgloss** - Styling and layout
- **Charm Log** - Structured logging
- **Glamour** - Markdown rendering
- **Huh** - Interactive forms

## Project Structure

```
klip-go/
├── main.go                    # Main entry point
├── cmd/klip/main.go          # CLI command implementation
├── internal/
│   ├── app/                  # Main application logic
│   ├── ui/                   # User interface components
│   │   ├── components/       # Reusable UI components
│   │   ├── styles/           # Styling and themes
│   │   └── pages/            # Different application pages
│   ├── api/                  # API clients and providers
│   │   └── providers/        # Provider-specific implementations
│   ├── storage/              # Data storage and encryption
│   └── utils/                # Utility functions
└── docs/                     # Documentation
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Building

```bash
# Build for current platform
make build

# Build for all platforms  
make build-all

# Run the application
make run
```

### Development Workflow

```bash
# Install dependencies
make deps

# Run quality checks
make check

# Run tests
make test

# Run in development mode (with live reload if air is installed)
make dev
```

## Phase 1 Status

This represents Phase 1 of the Go rewrite:

✅ **Complete Infrastructure Setup**
- Go module initialization with all Charm dependencies
- Complete project structure with proper package organization  
- Basic runnable application with Bubble Tea integration
- Cross-platform build system with Makefile
- Proper .gitignore and documentation

🚧 **Next Phase - Core Implementation**
- Full UI implementation with pages and navigation
- API client implementations for all providers
- Encrypted storage system
- Chat session management
- Streaming response handling

## Features (Planned)

- Multi-provider AI support (Anthropic, OpenAI, OpenRouter)
- Encrypted API key storage
- Real-time streaming responses
- Chat session logging
- Model switching and configuration
- Rich terminal interface with markdown rendering
- Cross-platform support (Linux, macOS, Windows)

## Build Targets

The application can be built for the following platforms:

- Linux (AMD64, ARM64)
- macOS (Intel, Apple Silicon)  
- Windows (AMD64, ARM64)