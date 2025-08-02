# CRUSH.md

## Build/Test/Lint Commands

```bash
deno task dev          # Run development server with watch mode
deno task build        # Build executable to ./dist/klip
deno task start        # Run application directly
deno task check        # Type check the codebase
deno task lint         # Lint source code
deno task fmt          # Format source code
deno run test-basic.ts # Run basic functionality test (if exists)
```

## Code Style Guidelines

### Imports & Formatting

- Use single quotes, semicolons, 2-space indentation (configured in deno.json)
- Import from JSR packages: `@std/encoding/hex`, `@cliffy/ansi/colors`
- Relative imports with `.ts` extension: `'./ui/app.ts'`
- Group imports: external packages first, then relative imports
- Use `type` imports for type-only imports: `import type { Model }`

### Types & Naming

- Use TypeScript strict mode with explicit interfaces
- PascalCase for classes: `ApiClient`, `KeyStore`
- camelCase for variables/methods: `currentModel`, `displayBanner()`
- Use descriptive interface names: `ChatRequest`, `Message`, `ResponseMetrics`

### Error Handling & Patterns

- Use try-catch blocks with descriptive error messages
- Implement retry logic with `withRetry()` utility
- Support interruption with `InterruptibleOperation`
- Log errors appropriately without exposing sensitive data
- Use async/await consistently, avoid Promise chains

