package cidr

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/artarts36/telegram-webhook-gateway/internal/config"
	"golang.org/x/net/proxy"
)

// Fetcher describes an abstraction for fetching a CIDR list.
type Fetcher interface {
	Fetch(ctx context.Context) ([]*net.IPNet, error)
}

// HTTPFetcher fetches CIDRs from a remote HTTP resource.
type HTTPFetcher struct {
	client *http.Client
	url    string
}

func NewHTTPFetcher(cfg config.TelegramConfig) (*HTTPFetcher, error) {
	httpClient, err := buildHTTPClient(cfg.Proxy.Value.Socks5.Address)
	if err != nil {
		return nil, fmt.Errorf("build http client: %w", err)
	}

	return &HTTPFetcher{
		client: httpClient,
		url:    cfg.CIDRURL,
	}, nil
}

func buildHTTPClient(socks5Address string) (*http.Client, error) {
	client := &http.Client{}
	if socks5Address == "" {
		return client, nil
	}

	dialer, err := proxy.SOCKS5("tcp", socks5Address, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("build socks5 dialer: %w", err)
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		IdleConnTimeout:       time.Minute,
		TLSHandshakeTimeout:   time.Minute,
		ExpectContinueTimeout: time.Minute,
	}

	if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
		transport.DialContext = contextDialer.DialContext
	} else {
		transport.DialContext = func(_ context.Context, network, address string) (net.Conn, error) {
			return dialer.Dial(network, address)
		}
	}

	client.Transport = transport

	return client, nil
}

// Fetch downloads and parses the list of CIDRs.
func (f *HTTPFetcher) Fetch(ctx context.Context) ([]*net.IPNet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req) //nolint:gosec // url got from trusted configuration
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseCIDRs(ctx, string(body))
}

// ParseCIDRs parses a list of CIDRs from text (one CIDR per line).
func ParseCIDRs(ctx context.Context, text string) ([]*net.IPNet, error) {
	lines := strings.Split(text, "\n")
	var nets []*net.IPNet
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		_, ipnet, err := net.ParseCIDR(line)
		if err != nil {
			slog.Default().WarnContext(ctx, "skip invalid CIDR", "cidr", line, "err", err)
			continue
		}
		nets = append(nets, ipnet)
	}
	return nets, nil
}

// HTTPError represents an HTTP request error with a non-OK status.
type HTTPError struct {
	StatusCode int
}

func (e *HTTPError) Error() string {
	return http.StatusText(e.StatusCode)
}
