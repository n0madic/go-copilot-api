# Repository Guidelines

## Project Structure & Module Organization
- `main.go` is the runtime entrypoint; it delegates to `internal/cli` subcommands.
- Core Go code lives in `internal/`:
  - `server/` HTTP handlers and routing
  - `github/`, `copilot/` upstream API clients
  - `anthropic/` request/response translation
  - `tokenizer/`, `sse/`, `state/`, `storage/`, `util/` shared internals
- Tests are colocated with the packages they validate (for example `internal/anthropic/*_test.go`, `internal/server/*_test.go`, `internal/sse/parser_test.go`).

## Build, Test, and Development Commands
- `go build ./...` — compile all packages.
- `go test ./...` — run the full test suite.
- `go test ./internal/sse -run TestReadEvents` — run SSE parser tests only.
- `go run ./main.go start --port 4141` — start the API server.
- `go run ./main.go auth` — run GitHub device auth flow.
- `go run ./main.go check-usage -json` — print full Copilot usage payload.
- `go run ./main.go debug --json` — print runtime/path diagnostics.

## Coding Style & Naming Conventions
- Format all Go code with `gofmt` before pushing.
- Keep files focused by responsibility (for CLI: one command per file where practical).
- Naming: exported identifiers in `PascalCase`, unexported in `camelCase`.
- Prefer explicit structs with JSON tags for API contracts.
- Keep logs actionable and concise; avoid noisy per-line debug output by default.

## Testing Guidelines
- Use the standard `testing` package.
- Name tests `TestXxx` and files `*_test.go`.
- Keep tests next to the package under test (for example `internal/server/count_tokens_test.go`).
- For HTTP behavior, use `httptest`; for upstream calls, mock via `http.RoundTripper`.

## Commit & Pull Request Guidelines
- This repository currently has no historical commits; adopt Conventional Commits:
  - `feat(cli): add check-usage -json`
  - `fix(server): preserve SSE flush behavior`
- PRs should include:
  - concise problem/solution summary
  - commands run (`go test ./...` at minimum)
  - sample output for changed CLI/API behavior
  - linked issue (if applicable)

## Security & Configuration Tips
- Tokens are stored at `~/.local/share/copilot-api/github_token`; keep permissions strict.
- Never commit secrets or local token files.
- Use `--show-token` only for local debugging and avoid sharing logs containing credentials.
