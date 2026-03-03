package gateway

import (
	"log/slog"
	"net/http"
)

type httpTransport struct {
	next http.RoundTripper
}

func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	slog.InfoContext(req.Context(), "[transport] sending request to target endpoint")

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		slog.WarnContext(req.Context(), "[transport] request to target endpoint failed", slog.Any("err", err))
		return resp, err
	}

	slog.InfoContext(req.Context(), "[transport] request to target endpoint was sent",
		slog.Int("http.status", resp.StatusCode),
	)

	return resp, nil
}
