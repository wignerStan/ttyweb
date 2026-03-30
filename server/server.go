package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	noesctmpl "text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"ttyweb/ai"
	"ttyweb/bindata"
	"ttyweb/db"
	"ttyweb/pkg/homedir"
	"ttyweb/pkg/randomstring"
	"ttyweb/service"
	"ttyweb/webtty"
)

// Server provides a webtty HTTP endpoint.
type Server struct {
	factory Factory
	options *Options

	upgrader        *websocket.Upgrader
	titleTemplate   *noesctmpl.Template
	noteSvc         *service.NoteService
	segmentService  *service.TaskSegmentService
	summaryService  *service.SummaryService
	stateMachine    *ai.StateMachine
	srvErrCh        chan error
	eventBus        *TaskEventBus
	sseHandler      *SSEHandler
	statsService    *service.StatsService
}

// indexHTML holds the SPA index.html content, loaded at init time.
var indexHTML []byte

// New creates a new instance of Server.
// Server will use the New() of the factory provided to handle each request.
func New(factory Factory, options *Options) (*Server, error) {
	indexData, err := bindata.Fs.ReadFile("static/index.html")
	if err != nil {
		panic("index not found in bindata") // must be in bindata
	}
	if options.IndexFile != "" {
		path := homedir.Expand(options.IndexFile)
		indexData, err = os.ReadFile(path) //nolint:gosec // reason: path comes from CLI option, not user input
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read custom index file at `%s`", path)
		}
	}
	indexHTML = indexData

	titleTemplate, err := noesctmpl.New("title").Parse(options.TitleFormat)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse window title format `%s`", options.TitleFormat)
	}

	var originChekcer func(r *http.Request) bool
	if options.WSOrigin != "" {
		matcher, err := regexp.Compile(options.WSOrigin)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compile regular expression of Websocket Origin: %s", options.WSOrigin)
		}
		originChekcer = func(r *http.Request) bool {
			return matcher.MatchString(r.Header.Get("Origin"))
		}
	} else {
		originChekcer = defaultOriginChecker
	}

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	noteSvc := service.NewNoteService(database)

	server := &Server{
		factory: factory,
		options: options,

		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    webtty.Protocols,
			CheckOrigin:     originChekcer,
		},
		titleTemplate:  titleTemplate,
		noteSvc:        noteSvc,
		segmentService: service.NewTaskSegmentService(database),
		stateMachine:   ai.NewStateMachine(),
		srvErrCh:       make(chan error, 1),
		eventBus:       NewTaskEventBus(),
	}
	server.sseHandler = NewSSEHandler(server.eventBus)

	registerAIStateChangeHandlers(server.stateMachine)

	return server, nil
}

// EventBus returns the server's TaskEventBus for publishing events.
func (s *Server) EventBus() *TaskEventBus {
	return s.eventBus
}

// Run starts the main process of the Server.
// The cancelation of ctx will shutdown the server immediately with aborting
// existing connections. Use WithGracefulContext() to support graceful shutdown.
func (server *Server) Run(ctx context.Context, options ...RunOption) error {
	cctx, cancel := context.WithCancel(ctx)
	opts := &RunOptions{gracefulCtx: context.Background()}
	for _, opt := range options {
		opt(opts)
	}

	counter := newCounter(time.Duration(server.options.Timeout) * time.Second)
	path := server.normalizePath()

	handlers := server.setupHandlers(cctx, cancel, path, counter)
	srv, err := server.setupHTTPServer(handlers)
	if err != nil {
		return errors.Wrapf(err, "failed to setup an HTTP server")
	}

	server.logStartupInfo()

	hostPort := net.JoinHostPort(server.options.Address, server.options.Port)
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", hostPort)
	if err != nil {
		return errors.Wrapf(err, "failed to listen at `%s`", hostPort)
	}

	server.logListenURLs(listener, path)

	go server.serveBackground(srv, listener)

	go func() {
		select {
		case <-opts.gracefulCtx.Done():
			_ = srv.Shutdown(context.Background())
		case <-cctx.Done():
		}
	}()

	select {
	case err = <-server.srvErrCh:
		if err == http.ErrServerClosed { // by graceful ctx
			err = nil
		} else {
			cancel()
		}
	case <-cctx.Done():
		_ = srv.Close()
		err = errors.Wrapf(cctx.Err(), "server context cancelled")
	}

	conn := counter.count()
	if conn > 0 {
		log.Printf("Waiting for %d connections to be closed", conn)
	}
	counter.wait()

	return err
}

// normalizePath returns the URL path prefix, ensuring it starts and ends with "/".
func (server *Server) normalizePath() string {
	path := server.options.Path
	if server.options.EnableRandomURL {
		path = "/" + randomstring.Generate(server.options.RandomURLLength) + "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

// logStartupInfo logs server configuration at startup.
func (server *Server) logStartupInfo() {
	if server.options.PermitWrite {
		log.Printf("Permitting clients to write input to the PTY.")
	}
	if server.options.Once {
		log.Printf("Once option is provided, accepting only one client")
	}
	if server.options.Port == "0" {
		log.Printf("Port number configured to `0`, choosing a random port")
	}
}

// logListenURLs logs the primary and alternative URLs after the listener is bound.
func (server *Server) logListenURLs(listener net.Listener, path string) {
	scheme := "http"
	if server.options.EnableTLS {
		scheme = "https"
	}
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	log.Printf("HTTP server is listening at: %s", scheme+"://"+net.JoinHostPort(host, port)+path)
	if server.options.Address == "0.0.0.0" {
		for _, address := range listAddresses() {
			log.Printf("Alternative URL: %s", scheme+"://"+net.JoinHostPort(address, port)+path)
		}
	}
}

// serveBackground starts the HTTP server (plain or TLS) in a goroutine.
func (server *Server) serveBackground(srv *http.Server, listener net.Listener) {
	var serveErr error
	if server.options.EnableTLS {
		crtFile := homedir.Expand(server.options.TLSCrtFile)
		keyFile := homedir.Expand(server.options.TLSKeyFile)
		log.Printf("TLS crt file: %s", crtFile)
		log.Printf("TLS key file: %s", keyFile)
		serveErr = srv.ServeTLS(listener, crtFile, keyFile)
	} else {
		serveErr = srv.Serve(listener)
	}
	if serveErr != nil {
		server.srvErrCh <- serveErr
	}
}

func (server *Server) setupHandlers(ctx context.Context, cancel context.CancelFunc, pathPrefix string, counter *counter) http.Handler {
	// Register the current working directory as the default file browser root.
	initFSDefaults()

	staticFS, err := fs.Sub(bindata.Fs, "static")
	if err != nil {
		log.Fatalf("failed to open static/ subdirectory of embedded filesystem: %v", err)
	}
	staticFileHandler := http.FileServer(http.FS(staticFS))

	var siteMux = http.NewServeMux()
	siteMux.HandleFunc(pathPrefix, server.handleIndex)
	// Serve SPA assets (Vite build output: assets/*.js, assets/*.css)
	siteMux.Handle(pathPrefix+"assets/", http.StripPrefix(pathPrefix, staticFileHandler))
	siteMux.Handle(pathPrefix+"favicon.ico", http.StripPrefix(pathPrefix, staticFileHandler))
	siteMux.Handle(pathPrefix+"icon.svg", http.StripPrefix(pathPrefix, staticFileHandler))

	siteHandler := http.Handler(siteMux)

	withGz := gziphandler.GzipHandler(server.wrapHeaders(siteHandler))
	siteHandler = server.wrapLogger(withGz)

	wsMux := http.NewServeMux()
	wsMux.Handle("/", siteHandler)
	wsMux.HandleFunc(pathPrefix+"ws", server.generateHandleWS(ctx, cancel, counter))
	wsMux.HandleFunc(pathPrefix+"ws/speech", server.handleSpeechWS)
	wsMux.HandleFunc(pathPrefix+"ws/ai/stream", server.handleAIStream)
	server.setupAPIHandlers(wsMux, pathPrefix)
	wsMux.HandleFunc(pathPrefix+"api/health", server.handleHealthCheck)
	siteHandler = http.Handler(wsMux)

	if server.options.EnableBasicAuth {
		log.Printf("Using Basic Authentication")
		siteHandler = server.wrapBasicAuth(siteHandler, server.options.Credential)
	}

	// Security middleware (wraps outer, executes before basic auth: csrf → cors → rateLimit → auth → handler)
	rateLimiter := newVisitorLimiter(10, 20) // 10 req/s per IP, burst 20
	siteHandler = rateLimitMiddleware(rateLimiter)(siteHandler)
	siteHandler = corsMiddleware(&CORSConfig{AllowedOrigins: server.options.CORSAllowedOrigins})(siteHandler)
	siteHandler = csrfMiddleware(siteHandler)

	return siteHandler
}

func (server *Server) setupHTTPServer(handler http.Handler) (*http.Server, error) {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if server.options.EnableTLSClientAuth {
		tlsConfig, err := server.tlsConfig()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to setup TLS configuration")
		}
		srv.TLSConfig = tlsConfig
	}

	return srv, nil
}

func (server *Server) tlsConfig() (*tls.Config, error) {
	caFile := homedir.Expand(server.options.TLSCACrtFile)
	caCert, err := os.ReadFile(caFile) //nolint:gosec // reason: caFile comes from CLI option, not user input
	if err != nil {
		return nil, errors.New("could not open CA crt file " + caFile)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("could not parse CA crt file data in " + caFile)
	}
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	return tlsConfig, nil
}

// defaultOriginChecker allows WebSocket connections when Origin is empty
// or when the Origin host matches the request host.
func defaultOriginChecker(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}
