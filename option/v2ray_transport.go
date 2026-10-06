package option

import (
	"strings"

	C "github.com/sagernet/sing-box/constant"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/json/badoption"
)

func NormalizeXHTTPMode(mode string) (string, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return "auto", nil
	}
	switch mode {
	case "auto", "packet-up", "stream-up", "stream-one":
		return mode, nil
	default:
		return "", E.New("unsupported mode: ", mode)
	}
}

type _V2RayTransportOptions struct {
	Type               string                  `json:"type"`
	HTTPOptions        V2RayHTTPOptions        `json:"-"`
	WebsocketOptions   V2RayWebsocketOptions   `json:"-"`
	QUICOptions        V2RayQUICOptions        `json:"-"`
	GRPCOptions        V2RayGRPCOptions        `json:"-"`
	HTTPUpgradeOptions V2RayHTTPUpgradeOptions `json:"-"`
	XHTTPOptions       V2RayXHTTPOptions       `json:"-"`
	KCPOptions         V2RayKCPOptions         `json:"-"`
}

type V2RayTransportOptions _V2RayTransportOptions

func (o V2RayTransportOptions) MarshalJSON() ([]byte, error) {
	var v any
	switch o.Type {
	case C.V2RayTransportTypeHTTP:
		v = o.HTTPOptions
	case C.V2RayTransportTypeWebsocket:
		v = o.WebsocketOptions
	case C.V2RayTransportTypeQUIC:
		v = o.QUICOptions
	case C.V2RayTransportTypeGRPC:
		v = o.GRPCOptions
	case C.V2RayTransportTypeHTTPUpgrade:
		v = o.HTTPUpgradeOptions
	case C.V2RayTransportTypeXHTTP:
		v = o.XHTTPOptions
	case "":
		return nil, E.New("missing transport type")
	default:
		return nil, E.New("unknown transport type: " + o.Type)
	}
	return badjson.MarshallObjects(_V2RayTransportOptions(o), v)
}

func (o *V2RayTransportOptions) UnmarshalJSON(bytes []byte) error {
	err := json.Unmarshal(bytes, (*_V2RayTransportOptions)(o))
	if err != nil {
		return err
	}
	var v any
	switch o.Type {
	case C.V2RayTransportTypeHTTP:
		v = &o.HTTPOptions
	case C.V2RayTransportTypeWebsocket:
		v = &o.WebsocketOptions
	case C.V2RayTransportTypeQUIC:
		v = &o.QUICOptions
	case C.V2RayTransportTypeGRPC:
		v = &o.GRPCOptions
	case C.V2RayTransportTypeHTTPUpgrade:
		v = &o.HTTPUpgradeOptions
	case C.V2RayTransportTypeXHTTP:
		v = &o.XHTTPOptions
	default:
		return E.New("unknown transport type: " + o.Type)
	}
	err = badjson.UnmarshallExcluded(bytes, (*_V2RayTransportOptions)(o), v)
	if err != nil {
		return err
	}
	return nil
}

type V2RayHTTPOptions struct {
	Host        badoption.Listable[string] `json:"host,omitempty"`
	Path        string                     `json:"path,omitempty"`
	Method      string                     `json:"method,omitempty"`
	Headers     badoption.HTTPHeader       `json:"headers,omitempty"`
	IdleTimeout badoption.Duration         `json:"idle_timeout,omitempty"`
	PingTimeout badoption.Duration         `json:"ping_timeout,omitempty"`
}

type V2RayWebsocketOptions struct {
	Path                string               `json:"path,omitempty"`
	Headers             badoption.HTTPHeader `json:"headers,omitempty"`
	MaxEarlyData        uint32               `json:"max_early_data,omitempty"`
	EarlyDataHeaderName string               `json:"early_data_header_name,omitempty"`
}

type V2RayQUICOptions struct{}

type V2RayGRPCOptions struct {
	ServiceName         string             `json:"service_name,omitempty"`
	IdleTimeout         badoption.Duration `json:"idle_timeout,omitempty"`
	PingTimeout         badoption.Duration `json:"ping_timeout,omitempty"`
	PermitWithoutStream bool               `json:"permit_without_stream,omitempty"`
	ForceLite           bool               `json:"-"` // for test
}

type V2RayHTTPUpgradeOptions struct {
	Host    string               `json:"host,omitempty"`
	Path    string               `json:"path,omitempty"`
	Headers badoption.HTTPHeader `json:"headers,omitempty"`
}

type V2RayKCPOptions struct {
	MTU              uint32 `json:"mtu,omitempty"`
	TTI              uint32 `json:"tti,omitempty"`
	UplinkCapacity   uint32 `json:"uplink_capacity,omitempty"`
	DownlinkCapacity uint32 `json:"downlink_capacity,omitempty"`
	Congestion       bool   `json:"congestion,omitempty"`
	ReadBufferSize   uint32 `json:"read_buffer_size,omitempty"`
	WriteBufferSize  uint32 `json:"write_buffer_size,omitempty"`
	HeaderType       string `json:"header_type,omitempty"`
	Seed             string `json:"seed,omitempty"`
	CwndMultiplier   uint32 `json:"cwnd_multiplier,omitempty"`
	MaxSendingWindow uint32 `json:"max_sending_window,omitempty"`
}

func (k *V2RayKCPOptions) GetMTU() uint32 {
	if k.MTU == 0 {
		return 1350
	}
	return k.MTU
}

func (k *V2RayKCPOptions) GetTTI() uint32 {
	if k.TTI == 0 {
		return 50
	}
	// Valid range: 10-5000 (extended from 10-100 to support high-latency networks)
	return k.TTI
}

func (k *V2RayKCPOptions) GetUplinkCapacity() uint32 {
	if k.UplinkCapacity == 0 {
		return 12
	}
	return k.UplinkCapacity
}

func (k *V2RayKCPOptions) GetDownlinkCapacity() uint32 {
	if k.DownlinkCapacity == 0 {
		return 100
	}
	return k.DownlinkCapacity
}

func (k *V2RayKCPOptions) GetReadBufferSize() uint32 {
	if k.ReadBufferSize == 0 {
		return 1
	}
	return k.ReadBufferSize
}

func (k *V2RayKCPOptions) GetWriteBufferSize() uint32 {
	if k.WriteBufferSize == 0 {
		return 1
	}
	return k.WriteBufferSize
}

func (k *V2RayKCPOptions) GetHeaderType() string {
	if k.HeaderType == "" {
		return "none"
	}
	return k.HeaderType
}

func (k *V2RayKCPOptions) GetCwndMultiplier() uint32 {
	if k.CwndMultiplier == 0 {
		return 20
	}
	return k.CwndMultiplier
}

func (k *V2RayKCPOptions) GetMaxSendingWindow() uint32 {
	return k.MaxSendingWindow
}
