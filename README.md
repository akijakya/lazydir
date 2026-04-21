# lazydir

A terminal user interface (TUI) for browsing and managing records in [AGNTCY Directory](https://github.com/agntcy/dir) instances — inspired by [lazygit](https://github.com/jesseduffield/lazygit) and [lazydocker](https://github.com/jesseduffield/lazydocker).

## Overview

`lazydir` lets you explore AGNTCY Directory nodes without memorizing `dirctl` commands. It presents the directory's contents in three navigable panels on the left side and a live preview panel on the right:

```
┌────────────────────────┬──────────────────────────────────────────────┐
│ [1] Directory          │ [Preview]                                     │
│  ● localhost:8888      │                                               │
│  c: connect            │  Shows either:                                │
├────────────────────────│  • OASF skill / domain / module description   │
│ [2] Classes            │  • Full record JSON (syntax highlighted)      │
│  Skills │ Domains │ Mod│                                               │
│  > natural_language…   │                                               │
│    text_to_code        │                                               │
│    …                   │                                               │
├────────────────────────│                                               │
│ [3] Records  /filter   │                                               │
│  > cisco.com/agent  v1 │                                               │
│    example.com/bot  v2 │                                               │
│    …                   │                                               │
└────────────────────────┴──────────────────────────────────────────────┘
  q:quit  tab:focus  ↑↓:nav  enter:select  /:filter  c:connect  r:refresh
```

### Panel descriptions

| Panel | Purpose |
|-------|---------|
| **[1] Directory** | Shows the current server address and connection status. Press `c` to open an inline dialog to connect to a different server. |
| **[2] Classes** | Displays taxonomy classes (Skills, Domains, Modules) aggregated from all records. Use `tab` to switch between the three tabs. Selecting a class fetches and displays its OASF description in the preview panel and filters the records list. |
| **[3] Records** | Lists all records (or a filtered subset when a class is selected). Shows name and version. Use `/` to filter by name. Press `enter` to load the full record JSON in the preview panel. |
| **Preview** | The right two-thirds of the screen. Displays either an OASF class description (plain text) or syntax-highlighted JSON of the selected record. Scroll with `↑`/`↓` when the preview panel is focused. |

## Prerequisites

- **Go 1.22+**
- A running [AGNTCY Directory](https://github.com/agntcy/dir) server (local daemon or remote)

To start a local daemon for testing:

```bash
dirctl daemon start
```

## Installation

```bash
go install github.com/akijakya/lazydir@latest
```

Or build from source using `go build`:

```bash
git clone https://github.com/akijakya/lazydir
cd lazydir
go build -o lazydir .
```

Or using [Task](https://taskfile.dev):

```bash
git clone https://github.com/akijakya/lazydir
cd lazydir
task build        # downloads deps and builds into .bin/lazydir
```

### Development workflow

| Command | Description |
|---------|-------------|
| `task deps` | Download Go module dependencies |
| `task build` | Build the binary into `.bin/lazydir` (incremental) |
| `task run` | Build and immediately start `lazydir` |
| `task run CLI_ARGS="--server-addr localhost:8888"` | Build and start with custom flags |

## Usage

```bash
lazydir [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--server-addr`, `-s` | `localhost:8888` | Directory server address |
| `--auth-mode`, `-a` | _(auto-detect)_ | Auth mode: `insecure`, `tls`, `oidc`, `jwt`, `x509` |
| `--auth-token` | | Pre-issued Bearer token (for CI / non-interactive) |
| `--tls-ca-file` | | TLS CA certificate file path |
| `--tls-cert-file` | | TLS client certificate file path |
| `--tls-key-file` | | TLS client key file path |
| `--tls-skip-verify` | `false` | Skip TLS certificate verification |
| `--help`, `-h` | | Show usage |

### Environment variables

| Variable | Description |
|----------|-------------|
| `DIRECTORY_CLIENT_SERVER_ADDRESS` | Default server address (overridden by `--server-addr`) |
| `DEBUG` | Set to any value to write a `lazydir_debug.log` file |

### Examples

```bash
# Connect to a local insecure server
lazydir --server-addr localhost:8888

# Connect with a pre-issued token
lazydir --server-addr my-dir.example.com:443 --auth-token "eyJ..."

# Connect using TLS certificates
lazydir -s my-dir.example.com:443 \
  --tls-ca-file /path/to/ca.pem \
  --tls-cert-file /path/to/client.crt \
  --tls-key-file /path/to/client.key
```

## Key Bindings

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | Quit |
| `tab` / `shift+tab` | Cycle panel focus |
| `1` | Focus the Directory panel |
| `2` | Focus the Classes panel |
| `3` | Focus the Records panel |
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `enter` | Select item (preview record / filter by class) |
| `esc` | Clear filter / dismiss dialog |
| `/` | Start name filter (Records panel) |
| `tab` (Classes panel) | Switch between Skills / Domains / Modules tabs |
| `c` (Directory panel) | Open connect dialog |
| `r` | Refresh records from server |
| `pgup` / `pgdown` | Scroll preview panel (when focused) |

## Architecture

```
lazydir/
├── main.go                        # Entry point; flag parsing; program startup
├── go.mod / go.sum
├── internal/
│   ├── app/
│   │   ├── model.go               # Root Bubble Tea model; layout; async command dispatch
│   │   ├── keys.go                # Global key constants and helpers
│   │   └── update.go              # (focus constants)
│   ├── panels/
│   │   ├── directory/             # Panel 1: server info + connect dialog
│   │   ├── classes/               # Panel 2: Skills / Domains / Modules tabs
│   │   └── records/               # Panel 3: record list + "/" filter
│   ├── preview/
│   │   └── preview.go             # Right viewport; JSON highlighting; text rendering
│   ├── dirclient/
│   │   └── wrapper.go             # Thin wrapper around github.com/agntcy/dir/client
│   └── oasf/
│       └── fetch.go               # HTTP fetch of OASF class descriptions; in-memory cache
```

### Data sources

| Data | Source |
|------|--------|
| Record list | `SearchRecords` gRPC call via `github.com/agntcy/dir/client` |
| Record JSON | `Pull` gRPC call (by CID) |
| Skill / Domain / Module descriptions | HTTP GET `https://schema.oasf.outshift.com/1.0.0/{skills|domains|modules}/{name}` |

### Technology

- **[Bubble Tea v2](https://charm.land/bubbletea/v2)** — TUI framework (Elm Architecture for Go)
- **[Lip Gloss v2](https://charm.land/lipgloss/v2)** — Terminal styling and layout
- **[Chroma v2](https://github.com/alecthomas/chroma)** — JSON syntax highlighting
- **[agntcy/dir client](https://github.com/agntcy/dir)** — gRPC client for Directory API

## Contributing

Pull requests and issues are welcome. Please open an issue first to discuss significant changes.

## License

Apache-2.0 — see [LICENSE](LICENSE).
