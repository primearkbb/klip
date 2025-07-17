# Klip - AI Chat TUI

A sleek terminal-based AI chat application built with Deno 2.4+ that supports multiple AI providers.

## Features

- **Multi-Provider Support**: Anthropic, OpenAI, and OpenRouter
- **Secure API Key Storage**: Encrypted storage using Deno's built-in crypto APIs
- **Model Switching**: Change models on-the-fly with smart autocomplete
- **Chat Management**: Clear history, edit and resend messages
- **Auto-Retry**: Automatic retry with exponential backoff for API errors
- **Interruption Handling**: Graceful Ctrl+C handling to stop responses
- **Chat Logging**: Structured local storage of chat sessions
- **Rich UI**: ASCII art banner and colorful terminal interface
- **Autocomplete**: Tab completion for model selection
- **Single Executable**: Compiles to a single-file executable
- **Cross-Platform**: Works on macOS and Linux

## Installation

### Prerequisites

- Deno 2.4+ installed on your system

### From Source

1. Clone the repository:
```bash
git clone <repository-url>
cd klip
```

2. Run the application:
```bash
deno task dev
```

3. Or build a single executable:
```bash
deno task build
```

## Usage

### First Run

On first run, you'll be prompted to enter API keys for the providers you want to use:

```
API key required for anthropic
Enter anthropic API key: [your-key-here]
```

### Basic Usage

- Type your message and press Enter or Ctrl+D to send
- Use `/help` to see available commands
- Press Ctrl+C to interrupt a response or exit

### Commands

- `/help` - Show help message
- `/models` - List available models
- `/model <model-id>` - Switch to a different model
- `/clear` - Clear chat history
- `/keys` - Manage API keys
- `/edit` - Edit and resend last message
- `/quit` - Exit the application

### Keyboard Shortcuts

- `Ctrl+C` - Interrupt current response or exit
- `Ctrl+D` - Send message
- `Ctrl+L` - Clear screen
- `Tab` - Autocomplete model names
- `↑/↓` - Navigate message history or autocomplete suggestions

## Configuration

### API Keys

API keys are stored encrypted in `~/.klip/keys.enc`. The encryption key is stored in `~/.klip/.key`.

### Chat Logs

Chat sessions are logged in `~/.klip/logs/` as JSON files with timestamps.

### Supported Models

**Anthropic:**
- Claude 3.5 Sonnet
- Claude 3.5 Haiku
- Claude 3 Opus

**OpenAI:**
- GPT-4 Turbo
- GPT-4o
- GPT-4o Mini
- GPT-3.5 Turbo

**OpenRouter:**
- Claude 3.5 Sonnet (via OpenRouter)
- GPT-4o (via OpenRouter)
- Llama 3.1 405B (via OpenRouter)

## Development

### Project Structure

```
src/
├── main.ts           # Entry point
├── ui/
│   ├── app.ts        # Main application logic
│   ├── banner.ts     # ASCII art and help
│   └── input.ts      # Terminal input handling
├── api/
│   ├── client.ts     # API client with retry logic
│   └── models.ts     # Model definitions
├── storage/
│   ├── keystore.ts   # Encrypted API key storage
│   └── logger.ts     # Chat logging
└── utils/
    └── retry.ts      # Retry and interruption utilities
```

### Building

```bash
# Development
deno task dev

# Build executable
deno task build

# The executable will be in ./dist/klip
```

### Security

- API keys are encrypted using AES-GCM with a 256-bit key
- Encryption key is stored with 600 permissions
- No sensitive data is logged in plain text

## License

MIT License - see LICENSE file for details