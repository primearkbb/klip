# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Klip is a terminal-based AI chat application built with Deno 2.4+ that supports multiple AI providers (Anthropic, OpenAI, OpenRouter). It features secure encrypted API key storage, model switching, chat logging, and a rich terminal interface.

## Development Commands

### Run Development Server
```bash
deno task dev
```

### Build Executable
```bash
deno task build
```
The executable will be created in `./dist/klip`

### Run Application
```bash
deno task start
```
or
```bash
./dist/klip  # if built
```

## Code Architecture

### Core Components

- **`src/main.ts`**: Entry point that initializes the App and displays banner
- **`src/ui/app.ts`**: Main application logic with chat loop, command handling, and user interaction
- **`src/api/client.ts`**: API client with retry logic supporting streaming responses for all providers
- **`src/api/models.ts`**: Model definitions and provider configurations
- **`src/storage/keystore.ts`**: Encrypted API key storage using AES-GCM
- **`src/storage/logger.ts`**: Chat session logging to JSON files
- **`src/utils/retry.ts`**: Retry utilities and interruption handling

### Key Features

- **Multi-provider support**: Handles different API formats (Anthropic vs OpenAI/OpenRouter)
- **Streaming responses**: Real-time chat with interruption support (Ctrl+C)
- **Encrypted storage**: API keys stored with AES-GCM encryption in `~/.klip/`
- **Command system**: Slash commands for model switching, chat management, etc.
- **Chat logging**: Sessions logged to `~/.klip/logs/` as timestamped JSON files

### Data Storage

- **Config directory**: `~/.klip/`
- **Encryption key**: `~/.klip/.key` (600 permissions)
- **Encrypted API keys**: `~/.klip/keys.enc`
- **Chat logs**: `~/.klip/logs/YYYY-MM-DD-HH-MM-SS.json`

### Provider Integration

Each provider (Anthropic, OpenAI, OpenRouter) has different API formats handled in `ApiClient`:
- Anthropic: `/v1/messages` endpoint with `system` parameter
- OpenAI: `/v1/chat/completions` with standard format
- OpenRouter: OpenAI-compatible format with additional headers

### UI Components

- **Banner**: ASCII art and help display
- **Input handling**: Simple stdin reading with history
- **Spinners**: Loading indicators for operations
- **Autocomplete**: Model selection interface

## Testing

Run basic functionality test:
```bash
deno run test-basic.ts
```

See `TESTING.md` for detailed testing procedures and expected behavior.

## Security Considerations

- API keys are encrypted using AES-GCM with 256-bit keys
- Config files have restricted permissions (600)
- No sensitive data logged in plain text
- Proper error handling prevents key exposure