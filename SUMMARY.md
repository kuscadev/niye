# niye — Implementation Summary

## Overview
A small CLI tool in Go that reads terminal error output and prints a two-line explanation:
- **Neden** (Cause): one sentence
- **Çözüm** (Fix): copy-pasteable command(s)

Command name: `niye`

---

## Files

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
| `cmd/niye/main_test.go`, `internal/*/ *_test.go` | Unit/integration tests for all components |
| `Makefile` | Build, test, vet, install targets |
| `LICENSE` | MIT license |
| `README.md` | English documentation (SUMMARY style) |
| `README_TR.md` | Turkish documentation, same depth |
| `.gitignore` | Build artifacts |
| `go.mod` | Module definition (`niye`, Go 1.27) |

---

## Features Implemented

### Usage
```bash
# Pipe stderr
go build ./... 2>&1 | niye

# Argument
niye "git: 'pushh' is not a git command."

# English output
niye --lang en "error message"

# Interactive config
niye init
```

### Output Format (exact)
```
● Neden: <one sentence>
● Çözüm:
    <command(s) the user can copy>
```

### Providers
1. **OpenAI-compatible** — configurable `base_url`, `model`, `api_key`
   - Covers OpenAI, OpenRouter, Ollama, Gemini OpenAI-compat endpoint
2. **Native Gemini** — `generateContent` with `systemInstruction`, JSON response,
   auth via `x-goog-api-key` header (never URL query)

### Config (`~/.config/niye/config.toml`, mode 0600)
```toml
provider = "openai"
base_url = "https://api.openai.com/v1"
model    = "gpt-4o-mini"
api_key  = "sk-..."
lang     = "tr"
timeout  = "30s"
```

**Precedence**: CLI flags > `NIYE_*` env vars > `OPENAI_API_KEY`/`GEMINI_API_KEY` > config file > defaults

### Privacy (masking before sending)
| Pattern | Replacement |
|---------|-------------|
| Home directory (`/home/user/...`) | `~/...` |
| Username (word boundaries, ≥3 chars) | `<user>` |
| `key=`, `token=`, `password=`, `secret=`, `passwd=` values (≥4 chars) | `***` |
| `sk-...`, `ghp_`/`gho_`/`github_pat_`, `AKIA...`, JWT, URL userinfo | `***` |
| `Bearer <token>` | `Bearer ***` |
| JSON `"key": "value"` | `"key": "***"` |

Only masked error text and a short system prompt are sent. Masking is best
effort and cannot guarantee every secret is caught. Model input is capped at
32 KB (head + tail kept, middle dropped).

### Safety
- **Never executes** suggested fix — only prints it
- **Warnings** for dangerous commands (printed above the command):
  - `rm -rf` targeting `/`, `~`, `$HOME`, `*` or `.`
  - `chmod -R 777` (with or without `sudo`)
  - `git push --force` / `-f`, `git reset --hard`
  - redirections into `/dev/sd*` / `/dev/nvme*`
  - `mkfs ...`
  - `dd of=...`
  - `curl ... | bash` / `wget ... | sh` patterns

### Error Handling & Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage/config error (missing key, bad config, empty input, invalid flag) |
| 2 | Provider error (401/403, 404, 429, network, timeout) |

---

## Tests
```bash
make check   # fmt + vet + test
```

**Test Coverage:**
- `internal/mask/mask_test.go`, `extra_test.go` — table-driven cases for every masking rule + negatives
- `internal/provider/provider_test.go`, `key_leak_test.go` — httptest servers: success, 401, 404, 429, malformed JSON, timeout, connection refused, Ollama no-auth, header auth, no key in errors
- `internal/config/config_test.go` — precedence matrix, TOML round-trip, 0600 mode, XDG_CONFIG_HOME
- `internal/render/render_test.go`, `dangerous_test.go` — exact golden strings, tr/en labels, dangerous command warnings
- `cmd/niye/main_test.go`, `input_limit_test.go` — E2E through `run()`: help, version, stdin, args, missing key, provider errors, init subcommand, 32 KB truncation, terminal usage
- `internal/secret/secret_test.go` — piped input, prompt rendering, terminal detection

---

## Key Design Decisions (from Plan)

| Decision | Implementation |
|----------|----------------|
| TOML parser | Hand-rolled minimal, zero deps (`internal/config`) |
| `niye init` | Writes config only, no live API verification |
| API key input | Hidden via `termios` ioctl (Linux/macOS), visible fallback |
| README | Split files (`README.md` EN + `README_TR.md` TR), SUMMARY style |
| Layout | `cmd/niye` + `internal/{config,mask,provider,render,i18n,secret}`, `module niye` |

---

## Verification
```bash
make check
```

---

## Example Run
```bash
$ ./niye "git: 'pushh' is not a git command."
● Neden: 'pushh' geçerli bir git komutu değil, muhtemelen 'push' yazılmak istenmiş.
● Çözüm:
    git push
```
