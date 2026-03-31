package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"ttyweb/backend/localcommand"
	"ttyweb/backend/tmux"
	"ttyweb/backend/zellij"
	"ttyweb/config"
	"ttyweb/db"
	"ttyweb/internal/slogutil"
	"ttyweb/server"
)

func main() {
	slog.SetDefault(slogutil.New(os.Stderr, "info"))

	var (
		configFile string
		addr       string
		port       string
		path       string
		backend    string
		cred       string
		enableTLS  bool
		tlsCrt     string
		tlsKey     string
		write      bool
		titleFmt   string
		session    string
		dbPath     string
	)

	flag.StringVar(&configFile, "config", "", "Path to JSON configuration file")
	flag.StringVar(&addr, "addr", "0.0.0.0", "IP address to listen")
	flag.StringVar(&port, "port", "8080", "Port number")
	flag.StringVar(&path, "path", "/", "Base path")
	flag.StringVar(&backend, "backend", "local", "Backend: local, tmux, zellij")
	flag.StringVar(&cred, "credential", "", "Basic auth credential (user:pass)")
	flag.BoolVar(&enableTLS, "tls", false, "Enable TLS")
	flag.StringVar(&tlsCrt, "tls-crt", "", "TLS certificate file")
	flag.StringVar(&tlsKey, "tls-key", "", "TLS key file")
	flag.BoolVar(&write, "w", false, "Permit client write")
	flag.StringVar(&titleFmt, "title-format", "{{ .command }}@ttyweb", "Window title format")
	flag.StringVar(&session, "session", "ttyweb", "Default session name (tmux/zellij)")
	flag.StringVar(&dbPath, "db", "", "SQLite database path (default: ~/.local/share/ttyweb/ttyweb.db)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ttyweb [options] [-- command args...]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nBackends:\n")
		fmt.Fprintf(os.Stderr, "  local  (default)  Raw shell command\n")
		fmt.Fprintf(os.Stderr, "  tmux             tmux session manager\n")
		fmt.Fprintf(os.Stderr, "  zellij           zellij terminal multiplexer\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  ttyweb -w bash\n")
		fmt.Fprintf(os.Stderr, "  ttyweb -backend tmux\n")
		fmt.Fprintf(os.Stderr, "  ttyweb -backend tmux -session mysession\n")
		fmt.Fprintf(os.Stderr, "  ttyweb -backend zellij -w\n")
	}
	flag.Parse()

	args := flag.Args()

	if err := loadConfig(configFile); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	dbPath = resolveDBPath(dbPath)
	if err := db.Init(dbPath); err != nil {
		slog.Warn("database initialization failed", "error", err)
	}

	options := buildOptions(addr, port, path, backend, cred, titleFmt, write, enableTLS, tlsCrt, tlsKey)

	factory, err := selectBackend(backend, session, args)
	if err != nil {
		slog.Error("failed to create backend", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(factory, options)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("database close failed", "error", err)
		}
	}()

	gracefulCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-gracefulCtx.Done()
		slog.Info("received signal, shutting down")
	}()

	if err := srv.Run(context.Background(), server.WithGracefulContext(gracefulCtx)); err != nil {
		slog.Error("server exited", "error", err)
	}
}

// loadConfig loads configuration from explicit path or default location.
func loadConfig(configFile string) error {
	configPath := configFile
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}
	_, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loadConfig: %w", err)
	}
	return nil
}

// resolveDBPath returns the SQLite database path, using defaults if empty.
func resolveDBPath(dbPath string) string {
	if dbPath == "" {
		dbOpts := db.DefaultOptions()
		return dbOpts.Path
	}
	return dbPath
}

// buildOptions constructs server.Options from CLI flags.
// Extracted from main() for testability.
func buildOptions(addr, port, path, backendName, cred, titleFmt string, write bool, enableTLS bool, tlsCrt, tlsKey string) *server.Options {
	options := &server.Options{
		Address:         addr,
		Port:            port,
		Path:            path,
		PermitWrite:     write,
		TitleFormat:     titleFmt,
		PermitArguments: backendName == "local",
		TitleVariables: map[string]any{
			"hostname": hostname(),
		},
	}

	// Credential: env var takes precedence over CLI flag
	if envCred := os.Getenv("TTYWEB_CREDENTIAL"); envCred != "" {
		cred = envCred
	} else if cred != "" {
		slog.Warn("credential flag is deprecated (visible in process list), use TTYWEB_CREDENTIAL environment variable instead")
	}
	if cred != "" {
		options.EnableBasicAuth = true
		options.Credential = cred
	}
	if enableTLS {
		options.EnableTLS = true
		if tlsCrt != "" {
			options.TLSCrtFile = tlsCrt
		}
		if tlsKey != "" {
			options.TLSKeyFile = tlsKey
		}
	}

	return options
}

// selectBackend creates a server.Factory for the named backend.
func selectBackend(backendName string, session string, args []string) (server.Factory, error) {
	switch backendName {
	case "local":
		if len(args) == 0 {
			args = []string{defaultShell()}
		}
		factory, err := localcommand.NewFactory(args[0], args[1:], &localcommand.Options{})
		if err != nil {
			return nil, fmt.Errorf("selectBackend: local factory: %w", err)
		}
		return factory, nil

	case "tmux":
		if !commandExists("tmux") {
			return nil, fmt.Errorf("tmux not found in PATH")
		}
		return tmux.NewFactory(session), nil

	case "zellij":
		if !commandExists("zellij") {
			return nil, fmt.Errorf("zellij not found in PATH")
		}
		return zellij.NewFactory(session), nil

	default:
		return nil, fmt.Errorf("unknown backend: %s", backendName)
	}
}

func defaultShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

func hostname() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "localhost"
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
