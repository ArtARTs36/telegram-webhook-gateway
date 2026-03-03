package gateway

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/artarts36/telegram-webhook-gateway/internal/config"
	"github.com/cappuccinotm/slogx/slogm"
	"github.com/google/uuid"
)

const (
	headerXRequestID = "X-Request-Id"

	defaultReadTimeout  = 15 * time.Second
	defaultWriteTimeout = 15 * time.Second
)

type IPChecker interface {
	Contains(ip net.IP) bool
}

// Server encapsulates the HTTP gateway server.
type Server struct {
	cfg        config.Config
	httpServer *http.Server
}

// NewServer creates and configures the HTTP server.
func NewServer(cfg config.Config, provider IPChecker) *Server {
	proxy := newReverseProxy(&cfg.Target.URL.Value)
	handler := newIPProtectedHandler(provider, proxy, cfg.IPHeaders)

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}

	return &Server{cfg: cfg, httpServer: srv}
}

// Run starts the HTTP server.
func (s *Server) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "starting telegram webhook gateway",
		"listen_addr", s.cfg.HTTPAddr,
		"target_url", s.cfg.Target.URL,
		"ip_headers", s.cfg.IPHeaders,
	)

	return s.httpServer.ListenAndServe()
}

func getOrGenerateRequestID(r *http.Request) string {
	requestID := r.Header.Get(headerXRequestID)
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return requestID
}

func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Route the request to the fixed upstream target.
			pr.SetURL(target)
			pr.SetXForwarded()

			// Preserve the target service Host header.
			pr.Out.Host = target.Host

			// Ensure X-Request-Id is always set on the outgoing request.
			requestID := getOrGenerateRequestID(pr.In)
			pr.Out.Header.Set(headerXRequestID, requestID)
		},
		Transport: &httpTransport{
			next: http.DefaultTransport,
		},
	}
}

func newIPProtectedHandler(checker IPChecker, proxy *httputil.ReverseProxy, headers []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get or generate a request ID for logging and forwarding.
		requestID := getOrGenerateRequestID(r)
		r.Header.Set(headerXRequestID, requestID)

		ctx := slogm.ContextWithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		slog.InfoContext(ctx, "handling request", slog.String("remote_addr", r.RemoteAddr))

		if r.Method != http.MethodPost {
			slog.WarnContext(ctx, "unallowed method", slog.String("remote_addr", r.RemoteAddr), slog.String("method", r.Method))
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		ip := getClientIP(r, headers)
		if ip == nil {
			slog.WarnContext(ctx, "unable to determine client IP", "remote_addr", r.RemoteAddr)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if !checker.Contains(ip) {
			slog.WarnContext(ctx, "forbidden request from ip", "ip", ip.String())
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Set the request ID header before proxying.
		r.Header.Set(headerXRequestID, requestID)

		proxy.ServeHTTP(w, r) // #nosec G704 - upstream target is fixed at startup and not user-controlled
	})
}
