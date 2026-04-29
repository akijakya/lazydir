# lazydir

A terminal user interface (TUI) for browsing and managing records in [AGNTCY Directory](https://github.com/agntcy/dir) instances — inspired by [lazygit](https://github.com/jesseduffield/lazygit) and [lazydocker](https://github.com/jesseduffield/lazydocker).

## Overview

`lazydir` lets you explore AGNTCY Directory nodes without memorizing `dirctl` commands. It presents the directory's contents in three navigable panels on the left side and a live preview panel on the right:

```
┌────────────────────────┬──────────────────────────────────────────────┐
│ [1] Connections        │ [Preview]                                    │
│  ● Directory: localh…  │                                              │
│  ● OASF: schema.oasf…  │  Shows either:                               │
├────────────────────────│  • OASF skill / domain / module description  │
│ [2] Filters            │  • Full record JSON (syntax highlighted)     │
│  > Skills              │                                              │
│      natural_language… │                                              │
│    Domains             │                                              │
│    Modules             │                                              │
│    OASF version        │                                              │
│    Version             │                                              │
│    Author              │                                              │
│    Trusted             │                                              │
│    Verified            │                                              │
├────────────────────────│                                              │
│ [3] Records  /filter   │                                              │
│  > cisco.com/agent  v1 │                                              │
│    example.com/bot  v2 │                                              │
│    …                   │                                              │
└────────────────────────┴──────────────────────────────────────────────┘
  navigate: ↑↓  focus: tab  filter records: /  open filter: enter
```

### Panel descriptions

| Panel | Purpose |
|-------|---------|
| **[1] Connections** | Shows both endpoints the TUI is currently talking to — the Directory server and the OASF schema server — along with the connection status of the former. Press `c` to switch to a different Directory server and `o` to point at a different OASF schema server. |
| **[2] Filters** | Lists every filter category (Skills, Domains, Modules, OASF version, Version, Author, Trusted, Verified). Pressing `enter` on a category opens its options view, where the values present in the loaded record set are listed; `tab` (or `enter`) toggles selection — multiple values can be active per category. `esc` returns from the options view to the filter list. Active selections appear as indented child rows under their category in the list view, and the `[3] Records` pane updates immediately as filters change. Press `/` to search within the filter list or within a category's options — results narrow live as you type. Press `i` in the options view to toggle the OASF class description inline (shown in green below the option). |
| **[3] Records** | Lists records that satisfy the active filters. Shows name and version. Use `/` to filter by name — results narrow live as you type. Press `enter` to load the full record JSON in the preview panel. Press `i` to toggle inline record info (CID, annotations, schema version, created-at) below the selected record. |
| **Preview** | The right two-thirds of the screen. Displays syntax-highlighted JSON of the selected record. Scroll with `↑`/`↓` when the preview panel is focused. |

## Prerequisites

- **Go 1.26+**
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
| `task fmt` | Format Go source files (`gofmt -s`) |
| `task vet` | Run `go vet` |
| `task lint` | Run `golangci-lint` (must be [installed](https://golangci-lint.run/welcome/install/)) |
| `task check` | Run fmt + vet + lint + build in one step |

## Usage

```bash
lazydir [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--server-addr`, `-s` | `localhost:8888` | Directory server address |
| `--oasf-addr`, `-o` | `https://schema.oasf.outshift.com` | OASF schema server URL (used via `oasf-sdk`) |
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
| `DIRECTORY_CLIENT_SERVER_ADDRESS` | Default Directory server address (overridden by `--server-addr`) |
| `OASF_SERVER_ADDRESS` | Default OASF schema server URL (overridden by `--oasf-addr`) |
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
| `1` | Focus the Connections panel |
| `2` | Focus the Filters panel |
| `3` | Focus the Records panel |
| `0` | Focus the Preview panel |
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `enter` (Filters list) | Open the options view for the selected category |
| `enter` (Filters options) | Toggle the option under the cursor |
| `tab` (Filters options) | Toggle the option under the cursor (multi-select) |
| `/` (Filters) | Live-search categories or options |
| `i` (Filters options) | Toggle inline OASF class description |
| `esc` (Filters) | Clear search query, or return to the filter list |
| `enter` (Records) | Load the full record JSON in the preview panel |
| `i` (Records) | Toggle inline record info (CID, annotations, schema version, created-at) |
| `/` (Records) | Live-filter by name |
| `esc` (Records) | Clear name filter |
| `c` (Connections panel) | Open Directory connect dialog |
| `o` (Connections panel) | Open OASF server connect dialog |
| `r` | Refresh records from server |
| `?` | Show the full keybinding popup for the focused panel |
| `wheel` | Scroll (list and preview panels) |

## Architecture

```
lazydir/
├── main.go                        # Entry point; flag parsing; program startup
├── go.mod / go.sum
├── internal/
│   ├── gui/
│   │   ├── gui.go                 # Top-level Gui struct; gocui init; async helpers
│   │   ├── layout.go              # Panel layout; frame drawing; status bar
│   │   ├── views.go               # Render functions for filters, records, and preview
│   │   ├── keybindings.go         # Key handlers; focus cycling; panel actions
│   │   ├── filters.go             # Filter state; category aggregation; query building
│   │   └── hints.go               # Options-bar and help-popup text generation
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
| Record info | `Pull` gRPC call (by CID), decoded to extract metadata |
| Skill / Domain / Module descriptions | OASF SDK schema client via `oasf-sdk/pkg/schema` |

### Technology

- **[gocui](https://github.com/jesseduffield/gocui)** — Terminal UI library (jesseduffield fork, as used by lazygit)
- **[Chroma v2](https://github.com/alecthomas/chroma)** — JSON syntax highlighting
- **[agntcy/dir client](https://github.com/agntcy/dir)** — gRPC client for Directory API

## Contributing

Pull requests and issues are welcome. Please open an issue first to discuss significant changes.

## License

Apache-2.0 — see [LICENSE](LICENSE).
