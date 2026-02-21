# Repository Guidelines

## Project Structure & Module Organization
- `main.go` is the entrypoint and delegates CLI handling to `internal/cli`.
- Main runtime packages are under `internal/`:
  - `server/` HTTP routing and handlers (`/v1/chat/completions`, `/v1/messages`, `/v1/responses`, etc.)
  - `responses/` OpenAI Responses API translation, streaming events, and in-memory response chaining store
  - `anthropic/` Anthropic ↔ OpenAI payload translation
  - `copilot/`, `github/` upstream API clients
  - `sse/`, `tokenizer/`, `state/`, `storage/`, `util/` shared infrastructure
  - `types/` API and transport contracts
- Tests are colocated with implementation (for example: `internal/server/*_test.go`, `internal/responses/*_test.go`, `internal/sse/parser_test.go`).

## Build, Test, and Development Commands
- `go build ./...` — compile all packages.
- `go test ./...` — run the full suite.
- `go test ./internal/server -run TestResponses` — run Responses API server tests.
- `go test ./internal/sse -run TestReadEvents` — run SSE parsing tests.
- `go run ./main.go start --bind 127.0.0.1 --port 4141` — start local API server.
- `go run ./main.go auth` — run GitHub device authorization.
- `go run ./main.go check-usage -json` — print full Copilot usage JSON.

## Coding Style & Naming Conventions
- Always run `gofmt` on changed Go files.
- Keep modules focused: handlers in `server/`, protocol translation in dedicated packages (`anthropic/`, `responses/`).
- Naming: exported identifiers `PascalCase`, internal helpers `camelCase`.
- Prefer explicit structs with stable JSON tags for API compatibility.
- Validate request invariants early (especially message/tool-call ordering) and fail with clear errors.

## Testing Guidelines
- Use Go `testing` with table-driven tests where useful.
- Name files `*_test.go`, functions `TestXxx`.
- For HTTP handlers use `httptest`; for upstream API behavior use mocked `http.RoundTripper`.
- Cover both non-stream and stream paths for protocol endpoints, including SSE event sequencing.

## Commit & Pull Request Guidelines
- Use Conventional Commits (examples: `feat(server): add responses stream events`, `fix(responses): normalize tool call ids`).
- PRs should include:
  - brief problem/solution summary
  - verification commands run (`go test ./...` minimum)
  - example request/response for changed API behavior
  - linked issue (if available)

## Security & Configuration Tips
- Token path: `~/.local/share/copilot-api/github_token`.
- Never commit secrets or token files.
- Use `--show-token` only for local debugging; do not share logs with credentials.
