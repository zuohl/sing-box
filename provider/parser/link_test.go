package parser

import (
	"encoding/base64"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"

	"github.com/stretchr/testify/require"
)

func TestParseVMessLinkSNI(t *testing.T) {
	link := "vmess://" + base64.RawURLEncoding.EncodeToString([]byte(`{"add":"192.0.2.1","port":"443","id":"11111111-1111-1111-1111-111111111111","tls":"tls","sni":"example.com"}`))
	outbound, err := ParseSubscriptionLink(link)
	require.NoError(t, err)

	options := outbound.Options.(*option.VMessOutboundOptions)
	require.Equal(t, "192.0.2.1", options.Server)
	require.Equal(t, "example.com", options.TLS.ServerName)
}

func TestParseVMessLinkHostServerNameFallback(t *testing.T) {
	testCases := []struct {
		name       string
		link       string
		serverName string
	}{
		{
			"websocket host",
			`{"add":"192.0.2.1","port":"443","id":"11111111-1111-1111-1111-111111111111","tls":"tls","net":"ws","host":"example.com"}`,
			"example.com",
		},
		{
			"http2 host",
			`{"add":"192.0.2.1","port":"443","id":"11111111-1111-1111-1111-111111111111","tls":"tls","net":"h2","host":"example.com"}`,
			"example.com",
		},
		{
			"explicit sni",
			`{"add":"192.0.2.1","port":"443","id":"11111111-1111-1111-1111-111111111111","tls":"tls","net":"ws","host":"host.example","sni":"sni.example"}`,
			"sni.example",
		},
		{
			"domain server",
			`{"add":"server.example","port":"443","id":"11111111-1111-1111-1111-111111111111","tls":"tls","net":"ws","host":"host.example"}`,
			"server.example",
		},
		{
			"grpc host",
			`{"add":"192.0.2.1","port":"443","id":"11111111-1111-1111-1111-111111111111","tls":"tls","net":"grpc","host":"example.com"}`,
			"192.0.2.1",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			link := "vmess://" + base64.RawURLEncoding.EncodeToString([]byte(testCase.link))
			outbound, err := ParseSubscriptionLink(link)
			require.NoError(t, err)

			options := outbound.Options.(*option.VMessOutboundOptions)
			require.Equal(t, testCase.serverName, options.TLS.ServerName)
		})
	}
}

func TestParseVLESSLinkHostServerNameFallback(t *testing.T) {
	const linkPrefix = "vless://11111111-1111-1111-1111-111111111111@"
	testCases := []struct {
		name       string
		link       string
		serverName string
	}{
		{
			"websocket host",
			"192.0.2.1:443?security=tls&type=ws&host=example.com",
			"example.com",
		},
		{
			"http host",
			"192.0.2.1:443?security=tls&type=http&host=example.com",
			"example.com",
		},
		{
			"explicit sni",
			"192.0.2.1:443?security=tls&type=ws&host=host.example&sni=sni.example",
			"sni.example",
		},
		{
			"domain server",
			"server.example:443?security=tls&type=ws&host=host.example",
			"server.example",
		},
		{
			"reality",
			"192.0.2.1:443?security=reality&type=ws&host=example.com",
			"192.0.2.1",
		},
		{
			"grpc host",
			"192.0.2.1:443?security=tls&type=grpc&host=example.com",
			"192.0.2.1",
		},
		{
			"host with port",
			"192.0.2.1:443?security=tls&type=ws&host=example.com%3A443",
			"192.0.2.1",
		},
		{
			"multiple http hosts",
			"192.0.2.1:443?security=tls&type=http&host=one.example%2Ctwo.example",
			"192.0.2.1",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			outbound, err := ParseSubscriptionLink(linkPrefix + testCase.link)
			require.NoError(t, err)

			options := outbound.Options.(*option.VLESSOutboundOptions)
			require.Equal(t, testCase.serverName, options.TLS.ServerName)
		})
	}
}

func TestParseV2RayLinkHostServerNameFallbackRequiresTLS(t *testing.T) {
	vmessLink := "vmess://" + base64.RawURLEncoding.EncodeToString([]byte(`{"add":"192.0.2.1","port":"443","id":"11111111-1111-1111-1111-111111111111","net":"ws","host":"example.com"}`))
	vmessOutbound, err := ParseSubscriptionLink(vmessLink)
	require.NoError(t, err)
	require.Nil(t, vmessOutbound.Options.(*option.VMessOutboundOptions).TLS)

	vlessOutbound, err := ParseSubscriptionLink("vless://11111111-1111-1111-1111-111111111111@192.0.2.1:443?type=ws&host=example.com")
	require.NoError(t, err)
	require.Nil(t, vlessOutbound.Options.(*option.VLESSOutboundOptions).TLS)
}

func TestParseHysteria2LinkOptions(t *testing.T) {
	outbound, err := ParseSubscriptionLink("hysteria2://password@192.0.2.1:443?sni=example.com&pinSHA256=AA:BB&mport=40000-50000")
	require.NoError(t, err)

	options := outbound.Options.(*option.Hysteria2OutboundOptions)
	require.Equal(t, []string{"40000:50000"}, []string(options.ServerPorts))
	require.Equal(t, "example.com", options.TLS.ServerName)
	require.Equal(t, "AA:BB", options.TLS.CertificatePinSHA256)
}

func TestParseVLESSLinkXHTTPAndECH(t *testing.T) {
	link := "vless://xxx@cdng.senflarelink.de5.net:443?encryption=none&type=xhttp&security=tls&path=%2F%3Fproxyip%3Dproxyip.zuohl.top%26ed%3D2560&host=xh.zuohldev.de5.net&mode=stream-one&extra=%7B+%0A++++%22xPaddingObfsMode%22%3A+true%2C+%0A++++%22xPaddingMethod%22%3A+%22tokenish%22%2C+%0A++++%22xPaddingPlacement%22%3A+%22queryInHeader%22%2C+%0A++++%22xPaddingHeader%22%3A+%22CqBCRk%22%2C+%0A++++%22xPaddingKey%22%3A+%22_CqBCRk%22+%0A++%7D&sni=xh.zuohldev.de5.net&alpn=h2&ech=lido.fi%2Bhttps%3A%2F%2F223.5.5.5%2Fdns-query#cdng"
	outbound, err := ParseSubscriptionLink(link)
	require.NoError(t, err)
	require.Equal(t, "cdng", outbound.Tag)

	options := outbound.Options.(*option.VLESSOutboundOptions)
	require.Equal(t, "cdng.senflarelink.de5.net", options.Server)
	require.Equal(t, uint16(443), options.ServerPort)
	require.Equal(t, "xxx", options.UUID)

	// TLS & ECH
	require.NotNil(t, options.TLS)
	require.True(t, options.TLS.Enabled)
	require.Equal(t, "xh.zuohldev.de5.net", options.TLS.ServerName)
	require.Equal(t, badoption.Listable[string]{"h2"}, options.TLS.ALPN)
	require.NotNil(t, options.TLS.ECH)
	require.True(t, options.TLS.ECH.Enabled)
	require.Equal(t, "lido.fi", options.TLS.ECH.QueryServerName)

	// XHTTP Transport
	require.NotNil(t, options.Transport)
	require.Equal(t, "xhttp", options.Transport.Type)
	require.Equal(t, "stream-one", options.Transport.XHTTPOptions.Mode)
	require.Equal(t, "xh.zuohldev.de5.net", options.Transport.XHTTPOptions.Host)
	require.Equal(t, "/?proxyip=proxyip.zuohl.top&ed=2560", options.Transport.XHTTPOptions.Path)
	require.True(t, options.Transport.XHTTPOptions.XPaddingObfsMode)
	require.Equal(t, "tokenish", options.Transport.XHTTPOptions.XPaddingMethod)
	require.Equal(t, "queryInHeader", options.Transport.XHTTPOptions.XPaddingPlacement)
	require.Equal(t, "CqBCRk", options.Transport.XHTTPOptions.XPaddingHeader)
	require.Equal(t, "_CqBCRk", options.Transport.XHTTPOptions.XPaddingKey)
}
