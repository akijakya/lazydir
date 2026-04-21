package main

import (
	"fmt"
	"os"

	"github.com/akijakya/lazydir/internal/dirclient"
	"github.com/akijakya/lazydir/internal/gui"
)

func main() {
	cfg := parseFlags()

	if err := gui.New(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func parseFlags() dirclient.Config {
	cfg := dirclient.Config{
		ServerAddress: "localhost:8888",
	}

	// Also honour the environment variable used by dirctl.
	if addr := os.Getenv("DIRECTORY_CLIENT_SERVER_ADDRESS"); addr != "" {
		cfg.ServerAddress = addr
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--server-addr", "-s":
			if i+1 < len(args) {
				i++
				cfg.ServerAddress = args[i]
			}
		case "--auth-mode", "-a":
			if i+1 < len(args) {
				i++
				cfg.AuthMode = args[i]
			}
		case "--auth-token":
			if i+1 < len(args) {
				i++
				cfg.AuthToken = args[i]
			}
		case "--tls-ca-file":
			if i+1 < len(args) {
				i++
				cfg.TLSCAFile = args[i]
			}
		case "--tls-cert-file":
			if i+1 < len(args) {
				i++
				cfg.TLSCertFile = args[i]
			}
		case "--tls-key-file":
			if i+1 < len(args) {
				i++
				cfg.TLSKeyFile = args[i]
			}
		case "--tls-skip-verify":
			cfg.TLSSkipVerify = true
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			printUsage()
			os.Exit(1)
		}
	}

	return cfg
}

func printUsage() {
	fmt.Fprint(os.Stderr, `lazydir - TUI for AGNTCY Directory

Usage:
  lazydir [flags]

Flags:
  --server-addr, -s   <addr>    Directory server address (default: localhost:8888)
  --auth-mode,   -a   <mode>    Auth mode: insecure|tls|oidc|jwt|x509 (default: auto-detect)
  --auth-token        <token>   Pre-issued Bearer token
  --tls-ca-file       <path>    TLS CA certificate file
  --tls-cert-file     <path>    TLS client certificate file
  --tls-key-file      <path>    TLS client key file
  --tls-skip-verify             Skip TLS certificate verification
  --help, -h                    Show this help

Environment:
  DIRECTORY_CLIENT_SERVER_ADDRESS  Default server address

Key Bindings (inside the TUI):
  tab / shift+tab    Cycle panel focus
  1 / 2 / 3          Jump to panel
  ↑ ↓ / j k          Navigate list items
  enter              Select item
  /                  Filter records by name
  esc                Clear filter / dismiss dialog
  h / l              Switch Classes tab (Skills / Domains / Modules)
  c                  Open connect dialog (Directory panel)
  r                  Refresh records
  q / ctrl+c         Quit
`)
}
