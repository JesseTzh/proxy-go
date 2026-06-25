package models

import "time"

type AuthConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Session struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"uniqueIndex;not null" json:"-"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type SystemSetting struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	InitialPortEnabled       bool       `json:"initialPortEnabled"`
	ManagementDomain         string     `json:"managementDomain"`
	ACMEEmail                string     `json:"acmeEmail"`
	RuntimeConfigStatus      string     `json:"runtimeConfigStatus"`
	SingBoxDebugEnabled      bool       `json:"singBoxDebugEnabled"`
	LastNginxReloadAt        *time.Time `json:"lastNginxReloadAt"`
	LastSingBoxRestartAt     *time.Time `json:"lastSingBoxRestartAt"`
	LastCertificateRenewalAt *time.Time `json:"lastCertificateRenewalAt"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
}

type ACMEChallenge struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Domain           string    `gorm:"index;not null" json:"domain"`
	Token            string    `gorm:"uniqueIndex;not null" json:"token"`
	KeyAuthorization string    `gorm:"type:text;not null" json:"-"`
	ExpiresAt        time.Time `json:"expiresAt"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Domain struct {
	ID            uint         `gorm:"primaryKey" json:"id"`
	Domain        string       `gorm:"uniqueIndex;not null" json:"domain"`
	CertificateID *uint        `json:"certificateId"`
	Certificate   *Certificate `json:"certificate,omitempty"`
	Status        string       `gorm:"not null;default:enabled" json:"status"`
	Remark        string       `json:"remark"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

type Certificate struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	PrimaryDomain string     `gorm:"index;not null" json:"primaryDomain"`
	SANDomains    string     `gorm:"type:text" json:"sanDomains"`
	Issuer        string     `json:"issuer"`
	CertFilePath  string     `json:"certFilePath"`
	KeyFilePath   string     `json:"keyFilePath"`
	ExpiresAt     *time.Time `json:"expiresAt"`
	AutoRenew     bool       `json:"autoRenew"`
	Status        string     `json:"status"`
	ErrorMessage  string     `gorm:"type:text" json:"errorMessage"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type ReverseProxyRule struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DomainID     uint      `gorm:"index;not null" json:"domainId"`
	Domain       Domain    `json:"domain"`
	TargetScheme string    `gorm:"not null" json:"targetScheme"`
	TargetHost   string    `gorm:"not null" json:"targetHost"`
	TargetPort   int       `gorm:"not null" json:"targetPort"`
	PreserveHost bool      `json:"preserveHost"`
	WebSocket    bool      `json:"webSocket"`
	PassRealIP   bool      `json:"passRealIp"`
	Enabled      bool      `json:"enabled"`
	Remark       string    `json:"remark"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ProxyInbound struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	Name                   string    `gorm:"not null" json:"name"`
	Template               string    `gorm:"index;not null" json:"template"`
	Protocol               string    `json:"protocol"`
	DomainID               uint      `gorm:"index;not null" json:"domainId"`
	Domain                 Domain    `json:"domain"`
	UUID                   string    `json:"-"`
	ListenAddr             string    `json:"listenAddr"`
	ListenPort             int       `gorm:"uniqueIndex" json:"listenPort"`
	Network                string    `json:"network"`
	Security               string    `json:"security"`
	Flow                   string    `json:"flow"`
	RouteSNI               string    `gorm:"index" json:"routeSni"`
	Password               string    `json:"-"`
	RealityHandshakeServer string    `json:"realityHandshakeServer"`
	RealityHandshakePort   int       `json:"realityHandshakePort"`
	RealityPrivateKey      string    `json:"-"`
	RealityPublicKey       string    `json:"-"`
	RealityShortID         string    `json:"-"`
	RealityMaxTimeDiff     int       `json:"realityMaxTimeDiff"`
	Enabled                bool      `json:"enabled"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Action       string    `gorm:"index;not null" json:"action"`
	ResourceType string    `gorm:"index" json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"userAgent"`
	CreatedAt    time.Time `gorm:"index" json:"createdAt"`
}

type ProtocolCapability struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex" json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
