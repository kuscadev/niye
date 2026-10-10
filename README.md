# niye

[Türkçe Dokümantasyon için tıklayın](README_TR.md)

## Overview

Small CLI tool in Go that reads terminal error output and prints a two-line explanation:

- **Cause**: one sentence
- **Fix**: copy-pasteable command(s)

Command name: `niye`. Requires Go 1.22+.

---

## Example

```bash
$ ./niye --lang en "git: 'pushh' is not a git command."
● Cause: 'pushh' is not a valid git command, likely a typo for 'push'.
● Fix:
    git push
```

Output format (exact):

```text
● Cause: <one sentence>
● Fix:
    <command(s) the user can copy>
```

---

## Installation

```bash
git clone https://github.com/kuscadev/niye.git
cd niye
make build          # builds `niye` binary
./niye --help       # test

make install        # install to ~/.local/bin
```

Or without cloning:

```bash
go install github.com/kuscadev/niye/cmd/niye@latest
```

---

## Quick Usage

```bash
# Pipe stderr
go build ./... 2>&1 | niye

# Argument
niye "git: 'pushh' is not a git command."

# Turkish output (default is tr)
niye --lang tr "error: command not found"

# Interactive config
niye init
```

```bash
niye [options] [error_message]

Options:
  --lang tr|en              Output language (default: tr)
  --provider openai|gemini  Provider (default: openai)
  --base-url URL            API base URL
  --model MODEL             Model name
  --api-key KEY             API key
  --timeout DURATION        Request timeout (default: 30s)
  --config FILE             Config file path
  -h, --help                Show this help
  --version                 Show version
```

No arguments and stdin is a terminal: prints usage and exits 1 without waiting.

Model input is capped at 32 KB: longer inputs keep the head and the tail, drop the middle, and print a one-line note to stderr.

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage/config error (missing key, bad config, empty input, invalid flag) |
| 2 | Provider error (401/403, 404, 429, network, timeout) |

---

## Configuration

Config file: `~/.config/niye/config.toml` (mode `0600`). Respects `XDG_CONFIG_HOME`.

```toml
provider = "openai"          # openai | gemini
base_url = "https://api.openai.com/v1"
model    = "gpt-4o-mini"
api_key  = "sk-..."          # optional; can use NIYE_API_KEY env var instead
lang     = "tr"              # tr | en
timeout  = "30s"
```

### Precedence

| Priority | Source |
|----------|--------|
| 1 (highest) | CLI flags (`--provider`, `--base-url`, `--model`, `--api-key`, `--lang`, `--timeout`, `--config`) |
| 2 | Environment variables (`NIYE_*`, then `OPENAI_API_KEY` / `GEMINI_API_KEY`) |
| 3 | `~/.config/niye/config.toml` |
| 4 (lowest) | Built-in defaults |

Environment variables:

- `NIYE_PROVIDER`, `NIYE_BASE_URL`, `NIYE_MODEL`, `NIYE_API_KEY`, `NIYE_LANG`, `NIYE_TIMEOUT`
- `OPENAI_API_KEY` (for openai provider)
- `GEMINI_API_KEY` (for gemini provider)

### Interactive Init

```bash
niye init
```

Prompts for: provider, base URL, model, **API key (hidden input on terminal)**, language, timeout.
Asks before overwriting existing config.
Offers option to **not store the key on disk** (uses `NIYE_API_KEY` env var instead).

---

## Providers

| Provider | Base URL Example | Auth |
|----------|------------------|------|
| **OpenAI** | `https://api.openai.com/v1` | `Bearer` header |
| **OpenRouter** | `https://openrouter.ai/api/v1` | `Bearer` header |
| **Ollama** | `http://localhost:11434/v1` | **No auth** (empty key) |
| **Gemini (OpenAI-compat)** | `https://generativelanguage.googleapis.com/v1beta/openai/` | `Bearer` header |
| **Gemini (native)** | `https://generativelanguage.googleapis.com/v1beta/models/<model-name>` | `x-goog-api-key` header |

For the current model name, see the provider docs.

Response handling: expects JSON `{"reason","fix"}`; falls back to labeled sections (`Cause:`/`Fix:`), fenced code blocks, then first-line-as-reason.

---

## Privacy

Before sending to the LLM, the error text is masked. Only the masked error text and a short system prompt are sent.

| Pattern | Replacement |
|---------|-------------|
| Home directory (`/home/user/...`) | `~/...` |
| Username (word boundaries, ≥3 chars) | `<user>` |
| `key=`, `token=`, `password=`, `secret=`, `passwd=` values (≥4 chars) | `***` |
| `sk-...` keys, `ghp_`/`gho_`/`github_pat_` tokens, `AKIA...` AWS IDs, JWT | `***` |
| `https://user...@host` userinfo | `https://***@host` |
| `Bearer <token>` | `Bearer ***` |
| JSON `"key": "value"` | `"key": "***"` |

Masking is best effort: it cannot guarantee every secret is caught. Review sensitive output before sharing.

Note: on the free Gemini tier, Google may use content for product development. For sensitive content, use a local provider (e.g. Ollama) or a paid plan.

Model input is capped at 32 KB (head + tail kept, middle dropped).

---

## Safety

- **The suggested fix is NEVER executed.** Only printed.
- Dangerous commands trigger a warning line **above** the command:
  - `rm -rf` targeting `/`, `~`, `$HOME`, `*` or `.`
  - `chmod -R 777` (with or without `sudo`)
  - `git push --force` / `-f`, `git reset --hard`
  - Redirections into `/dev/sd*` / `/dev/nvme*` devices
  - `mkfs ...`
  - `dd of=...`
  - `curl ... | bash` / `wget ... | sh` patterns
- Output is colorless and pipe-safe.

---

## Architecture

```text
stdin / argv (error text)
        │
        ▼
┌───────────────┐
│ cmd/niye      │  flag parsing, run, runInit
└───┬───┬───┬───┘
    │   │   │
    ▼   ▼   ▼
 config mask i18n        secret (init: hidden API-key input)
    │     │
    │     ▼  masked error text (max 32 KB) + system prompt
    │   ┌──────────┐
    │   │ provider │  OpenAI-compat / native Gemini
    │   └──────────┘
    │         │
    ▼         ▼
  config   answer {reason, fix}
  file            │
                  ▼
            ┌──────────┐
            │ render   │  two-line output + danger warnings
            └──────────┘
```

| Path | Purpose |
|------|---------|
| `cmd/niye/main.go` | Entry point, flag parsing, main logic, subcommand handling |
| `internal/config/config.go` | Config loading (TOML, env vars, CLI flags), precedence, save with 0600 |
| `internal/mask/mask.go` | Privacy masking (home dir, username, secrets, Bearer tokens) |
| `internal/provider/provider.go` | Provider interface, shared HTTP, JSON/plain-text extraction |
| `internal/provider/provider_openai.go` | OpenAI-compatible adapter (OpenAI, OpenRouter, Ollama, Gemini compat) |
| `internal/provider/provider_gemini.go` | Native Gemini API adapter |
| `internal/render/render.go` | Two-line output format, dangerous command warnings |
| `internal/i18n/i18n.go` | Turkish/English message tables |
| `internal/secret/secret.go` | Secure API key input (platform-agnostic interface) |
| `internal/secret/secret_linux.go` | Linux termios echo suppression |
| `internal/secret/secret_darwin.go` | macOS termios echo suppression |
| `internal/secret/secret_other.go` | Fallback for unsupported platforms |
| `LICENSE` | MIT license |
| `go.mod` | Module definition (`github.com/kuscadev/niye`, Go 1.22) |

---

## Technical Decisions

| Decision | Implementation |
|----------|----------------|
| TOML parser | Hand-rolled minimal, zero deps (`internal/config`) |
| `niye init` | Writes config only, no live API verification |
| API key input | Hidden via `termios` ioctl (Linux/macOS), visible fallback |
| Gemini auth | `x-goog-api-key` header; key never placed in the URL so error messages can't leak it |
| Input cap | 32 KB head+tail truncation with one-line stderr note |
| Response parsing | Strict JSON first, then balanced-brace scan, labeled text, code fence, first-line fallback |
| Layout | `cmd/niye` + `internal/{config,mask,provider,render,i18n,secret}` |

---

## Development & Tests

```bash
make check     # fmt + vet + test
make test      # run all tests
make vet       # go vet
make fmt       # gofmt
```

### Test Coverage

| Test file | Scope |
|-----------|-------|
| `internal/mask/mask_test.go`, `extra_test.go` | Table-driven cases for every masking rule + negatives |
| `internal/provider/provider_test.go`, `key_leak_test.go` | httptest servers: success, 401, 404, 429, malformed JSON, timeout, connection refused, Ollama no-auth, header auth, no key in errors |
| `internal/config/config_test.go` | Precedence matrix, TOML round-trip, 0600 mode, XDG_CONFIG_HOME |
| `internal/render/render_test.go`, `dangerous_test.go` | Exact golden strings, tr/en labels, dangerous command warnings |
| `cmd/niye/main_test.go`, `input_limit_test.go` | E2E through `run()`: help, version, stdin, args, missing key, provider errors, init subcommand, 32 KB truncation, terminal usage |
| `internal/secret/secret_test.go` | Piped input, prompt rendering, terminal detection |

---

## License

MIT License — see `LICENSE` file.
