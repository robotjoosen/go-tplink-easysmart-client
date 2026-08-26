package model

type Device struct {
	Description     string `json:"description"`
	MAC             string `json:"mac"`
	IP              string `json:"ip"`
	SubnetMask      string `json:"subnet_mask"`
	DefaultGateway  string `json:"default_gateway"`
	FirmwareVersion string `json:"firmware_version"`
	HardwareVersion string `json:"hardware_version"`
}

type Port struct {
	Number int    `json:"number"`
	Label  string `json:"label"`
	Stats  Stats  `json:"stats"`
}

type Stats struct {
	LinkStatus       string  `json:"link_status"`
	LinkStatusNumber int     `json:"link_status_number"`
	Enabled          bool    `json:"enabled"`
	Packets          Packets `json:"packets"`
}

type Packets struct {
	Transmitter Transfer `json:"transmitter"`
	Receiver    Transfer `json:"receiver"`
}

type Transfer struct {
	Good int `json:"good"`
	Bad  int `json:"bad"`
}

type PortSpeed struct {
	PortNumber      int `json:"port_number"`
	IngressRateKbps int `json:"ingress_rate_kbps"`
	EgressRateKbps  int `json:"egress_rate_kbps"`
}

type CableDiagnostics struct {
	PortNumber       int    `json:"port_number"`
	CableState       string `json:"cable_state"`
	CableStateNumber int    `json:"cable_state_number"`
	Length           int    `json:"length"`
}
