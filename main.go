package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"ttyweb/backend/localcommand"
	"ttyweb/backend/tmux"
	"ttyweb/backend/zellij"
	"ttyweb/config"
	"ttyweb/db"
	"ttyweb/server"
)

func main() {
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

	// Load configuration: explicit path, or default location.
	if configFile != "" {
		_, err := config.Load(configFile)
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
	}
	config.LoadOrDefault()

	// Initialize database.
	if dbPath == "" {
		dbOpts := db.DefaultOptions()
		dbPath = dbOpts.Path
	}
	if err := db.Init(dbPath); err != nil {
		log.Printf("warning: database initialization failed: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("warning: database close failed: %v", err)
		}
	}()

	options := &server.Options{
		Address:             addr,
		Port:                port,
		Path:                path,
		PermitWrite:         write,
		TitleFormat:         titleFmt,
		PermitArguments:     backend == "local",
		TitleVariables: map[string]interface{}{
			"hostname": hostname(),
		},
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

	var factory server.Factory
	var err error

	switch backend {
	case "local":
		if len(args) == 0 {
			args = []string{defaultShell()}
		}
		factory, err = localcommand.NewFactory(args[0], args[1:], &localcommand.Options{})

	case "tmux":
		if !commandExists("tmux") {
			log.Fatal("tmux not found in PATH")
		}
		factory = tmux.NewFactory(session)

	case "zellij":
		if !commandExists("zellij") {
			log.Fatal("zellij not found in PATH")
		}
		factory = zellij.NewFactory(session)

	default:
		log.Fatalf("unknown backend: %s", backend)
	}

	if err != nil {
		log.Fatalf("failed to create backend: %v", err)
	}

	srv, err := server.New(factory, options)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received signal, shutting down...")
		cancel()
	}()

	if err := srv.Run(ctx, server.WithGracefullContext(context.Background())); err != nil {
		log.Printf("Server exited: %v", err)
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
