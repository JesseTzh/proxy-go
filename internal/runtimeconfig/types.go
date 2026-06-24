package runtimeconfig

type Snapshot struct {
	ManagementDomain string
	PublicHTTPSPort  int
	ManagedHTTPSAddr string
	ReverseProxies   []ReverseProxy
	ProxyInbounds    []ProxyInbound
}

type ReverseProxy struct {
	Domain       string
	TargetScheme string
	TargetHost   string
	TargetPort   int
	PreserveHost bool
	WebSocket    bool
	PassRealIP   bool
}

type ProxyInbound struct {
	ID                     uint
	Name                   string
	Template               string
	PublicHTTPSPort        int
	ManagedHTTPSAddr       string
	Protocol               string
	Domain                 string
	ListenAddr             string
	ListenPort             int
	UUID                   string
	Network                string
	Security               string
	Flow                   string
	XHTTPPath              string
	XHTTPMode              string
	RealityPrivateKey      string
	RealityPublicKey       string
	RealityShortID         string
	RealityHandshakeServer string
	RealityHandshakePort   int
	RealityMaxTimeDiff     int
}
