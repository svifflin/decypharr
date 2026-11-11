package manager

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirrobot01/decypharr/internal/config"
	debrid "github.com/sirrobot01/decypharr/pkg/debrid/common"
)

type streamClientProvider struct {
	debrid.Client
	conf config.Debrid
}

func (p streamClientProvider) Config() config.Debrid { return p.conf }

func TestStreamClientForUsesProviderProxy(t *testing.T) {
	base := &http.Client{}
	m := &Manager{
		streamClient:        base,
		clients:             xsync.NewMap[string, debrid.Client](),
		debridStreamClients: xsync.NewMap[string, *http.Client](),
	}
	m.clients.Store("AllDebrid", streamClientProvider{conf: config.Debrid{
		Name:  "AllDebrid",
		Proxy: "http://gluetun:8888",
	}})
	m.clients.Store("RealDebrid", streamClientProvider{conf: config.Debrid{Name: "RealDebrid"}})

	proxied := m.streamClientFor("AllDebrid")
	if proxied == base {
		t.Fatal("proxied provider must not share the environment stream client")
	}
	if got := proxyForURL(t, proxied, "https://alldebrid.com/f/abc"); got != "http://gluetun:8888" {
		t.Fatalf("proxy = %q, want %q", got, "http://gluetun:8888")
	}
	if again := m.streamClientFor("AllDebrid"); again != proxied {
		t.Fatal("stream client should be cached per provider")
	}

	if direct := m.streamClientFor("RealDebrid"); direct != base {
		t.Fatal("provider without a proxy must use the shared stream client")
	}
	if unknown := m.streamClientFor("Unknown"); unknown != base {
		t.Fatal("unknown provider must use the shared stream client")
	}
}

func TestNewStreamClientRoutesThroughProxy(t *testing.T) {
	var gotHost string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(proxy.Close)

	resp, err := newStreamClient(proxy.URL).Get("http://media.invalid/file.mkv")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if gotHost != "media.invalid" {
		t.Fatalf("proxy received host %q, want %q", gotHost, "media.invalid")
	}
}

func proxyForURL(t *testing.T, client *http.Client, rawURL string) string {
	t.Helper()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if proxyURL == nil {
		return ""
	}
	return proxyURL.String()
}
