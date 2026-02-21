# go-copilot-api

A Go server that proxies GitHub Copilot as an OpenAI/Anthropic-compatible API.

> This project is a **Go port** of the original project: https://github.com/ericc-ch/copilot-api

## Features

- OpenAI-compatible endpoints:
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
  - `GET /v1/models`
  - `POST /v1/embeddings`
- Anthropic-compatible endpoints:
  - `POST /v1/messages`
  - `POST /v1/messages/count_tokens`
- CLI commands:
  - `start` — start the server
  - `auth` — run GitHub OAuth device flow
  - `check-usage` — inspect Copilot quotas
  - `debug` — print diagnostic information

## Requirements

- Go 1.23+
- A GitHub account with an active Copilot subscription

## Quick Start

```bash
go test ./...
go run ./main.go auth
go run ./main.go start --port 4141
```

After startup, the server is available at `http://127.0.0.1:4141`.

## Useful Commands

```bash
# Run all tests
go test ./...

# Check usage in human-readable format
go run ./main.go check-usage

# Print full usage payload as JSON
go run ./main.go check-usage -json

# Diagnostics
go run ./main.go debug --json
```

## Responses API Notes

- `POST /v1/responses` supports both:
  - non-stream responses (`stream=false` or omitted)
  - SSE streaming (`stream=true`)
- `previous_response_id` is supported with an in-memory conversation store
  (state is process-local and is not persisted across restarts).

## CLI Flags

### `start`

- `--port <int>`: server port (default: `4141`)
- `--bind <addr>`: bind address (default: `127.0.0.1`, e.g. `0.0.0.0`, `::`)
- `--verbose`: enable verbose logs
- `--account-type <individual|business|enterprise>`: Copilot account type (default: `individual`)
- `--rate-limit <seconds>`: minimum delay between requests
- `--wait`: wait instead of returning rate-limit error
- `--github-token <token>`: use pre-generated GitHub token directly
- `--claude-code`: print a ready-to-run Claude Code command
- `--show-token`: print GitHub/Copilot tokens in logs (debug use only)
- `--proxy-env`: use proxy settings from environment (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`)

### `auth`

- `--verbose`: enable verbose logs
- `--show-token`: print fetched GitHub token in logs

### `check-usage`

- `--json` (or `-json`): print full `GetCopilotUsage` response as pretty JSON

### `debug`

- `--json`: print diagnostics as JSON

## Token Storage

The GitHub token is stored at:

```text
~/.local/share/copilot-api/github_token
```

The file is created automatically during `auth`/`start`.

## Important Notes

- This project relies on reverse-engineered Copilot APIs; upstream behavior may change.
- Do not publish tokens or logs containing sensitive data.
- See `AGENTS.md` for contributor/development guidelines.
