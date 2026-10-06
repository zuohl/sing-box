package v2rayxhttp

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
)

func TestNewClientHTTP3(t *testing.T) {
	tlsConfig, err := tls.NewClient(context.Background(), nil, "example.com", option.OutboundTLSOptions{
		Enabled:    true,
		ServerName: "example.com",
		ALPN:       []string{"h3"},
	})
	if err != nil {
		t.Fatalf("NewClient TLS: %v", err)
	}
	opts := option.V2RayXHTTPOptions{
		Mode: "stream-one",
		Path: "/xhttp",
	}
	transport, err := NewClient(context.Background(), nil, M.ParseSocksaddr("127.0.0.1:443"), opts, tlsConfig)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c := transport.(*Client)
	conn := c.xmux.newConn()
	if _, ok := conn.(*http3XmuxConn); !ok {
		t.Fatalf("expected *http3XmuxConn, got %T", conn)
	}
}

func TestURLQueryInPath(t *testing.T) {
	opts := option.V2RayXHTTPOptions{
		Mode: "stream-one",
		Path: "/?proxyip=proxyip.zuohl.top&ed=2560",
	}
	tlsConfig, err := tls.NewClient(context.Background(), nil, "example.com", option.OutboundTLSOptions{
		Enabled:    true,
		ServerName: "example.com",
		ALPN:       []string{"h3"},
	})
	if err != nil {
		t.Fatalf("tls: %v", err)
	}
	c, err := NewClient(context.Background(), nil, M.ParseSocksaddr("127.0.0.1:443"), opts, tlsConfig)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	req, err := c.(*Client).newRequest(context.Background(), "POST", "", "", nil)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	if req.URL.Path != "/" {
		t.Fatalf("want Path '/', got %q", req.URL.Path)
	}
	if req.URL.RawQuery != "proxyip=proxyip.zuohl.top&ed=2560" {
		t.Fatalf("want RawQuery 'proxyip=proxyip.zuohl.top&ed=2560', got %q", req.URL.RawQuery)
	}
	if req.URL.RequestURI() != "/?proxyip=proxyip.zuohl.top&ed=2560" {
		t.Fatalf("want RequestURI '/?proxyip=proxyip.zuohl.top&ed=2560', got %q", req.URL.RequestURI())
	}
}
