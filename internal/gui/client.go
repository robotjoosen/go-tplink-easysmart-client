package gui

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
	"github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
)

const (
	PathLogin      = "/logon.cgi"
	PathSystemInfo = "/SystemInfoRpm.htm"
	PathPortStats  = "/PortStatisticsRpm.htm"
	PathLedControl = "/led_on_set.cgi"
	PathClearStats = "/port_statistics_set.cgi"
	PathQoS        = "/QosBandWidthControlRpm.htm"
	PathQoSSet     = "/qos_bandwidth_set.cgi"
	PathCableDiag  = "/cable_diag_get.cgi"
)

type Client struct {
	client  *http.Client
	address string
}

func NewClient(address string) *Client {
	jar, _ := cookiejar.New(nil)

	return &Client{
		client:  &http.Client{Jar: jar},
		address: "http://" + address,
	}
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	loginUrl := c.address + PathLogin

	formValues := url.Values{}
	formValues.Add("username", username)
	formValues.Add("password", password)
	formValues.Add("cpassword", "")
	formValues.Add("logon", "Login")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginUrl, strings.NewReader(formValues.Encode()))
	if err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rsp, err := c.client.Do(req)
	if err != nil {
		return errors.Join(err, clientErrors.ErrAuthenticationFailed)
	}

	if rsp.StatusCode != http.StatusOK {
		return errors.Join(err, clientErrors.ErrAuthenticationFailed)
	}

	return nil
}

func (c *Client) GetDevice(ctx context.Context) (model.Device, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathSystemInfo, nil)
	if err != nil {
		return model.Device{}, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return model.Device{}, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return model.Device{}, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	html := string(bodyBytes)
	deviceInfo := parseDeviceInfo(html)
	if len(deviceInfo) == 6 {
		return model.Device{}, clientErrors.ErrInvalidResponse
	}

	return model.Device{
		Description:     deviceInfo[0],
		IP:              deviceInfo[1],
		MAC:             deviceInfo[2],
		SubnetMask:      deviceInfo[3],
		DefaultGateway:  deviceInfo[4],
		FirmwareVersion: deviceInfo[5],
		HardwareVersion: deviceInfo[6],
	}, nil
}

func (c *Client) GetPortStats(ctx context.Context) ([]model.Port, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathPortStats, nil)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return []model.Port{}, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return []model.Port{}, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	html := string(bodyBytes)
	maxPorts := parseMaxPorts(html)
	packets := parsePackets(html, maxPorts)
	linkLabels := parseLinkLabels(html, maxPorts)
	linkStatus := parseLinkStatus(html, maxPorts)
	status := parseStatusNumber(html, maxPorts)

	ports := make([]model.Port, maxPorts)
	for i := range ports {
		ports[i] = model.Port{
			Number: i + 1,
			Label:  "port " + strconv.Itoa(i+1),
			Stats: model.Stats{
				Enabled:          status[i] != 0,
				LinkStatus:       linkLabels[linkStatus[i]],
				LinkStatusNumber: linkStatus[i],
				Packets: model.Packets{
					Transmitter: model.Transfer{
						Good: packets[i][0],
						Bad:  packets[i][1],
					},
					Receiver: model.Transfer{
						Good: packets[i][2],
						Bad:  packets[i][3],
					},
				},
			},
		}
	}

	return ports, err
}

func (c *Client) GetCableDiagnostics(ctx context.Context) ([]model.CableDiagnostics, error) {
	maxPorts, err := c.getMaxPorts(ctx)
	if err != nil {
		return nil, err
	}

	if maxPorts == 0 {
		return nil, clientErrors.ErrInvalidResponse
	}

	p := url.Values{}
	for i := 1; i <= maxPorts; i++ {
		p.Add("chk_"+strconv.Itoa(i), strconv.Itoa(i))
	}
	p.Add("Apply", "Apply")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathCableDiag+"?"+p.Encode(), nil)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	html := string(bodyBytes)
	states := parseCableState(html, maxPorts)
	lengths := parseCableLength(html, maxPorts)
	if len(states) == 0 || len(lengths) == 0 {
		return nil, clientErrors.ErrInvalidResponse
	}

	diagnostics := make([]model.CableDiagnostics, maxPorts)
	for i := range diagnostics {
		diagnostics[i] = model.CableDiagnostics{
			PortNumber:       i + 1,
			CableState:       cableStateLabel(states[i]),
			CableStateNumber: states[i],
			Length:           lengths[i],
		}
	}

	return diagnostics, nil
}

func (c *Client) getMaxPorts(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathPortStats, nil)
	if err != nil {
		return 0, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return 0, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return 0, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	return parseMaxPorts(string(bodyBytes)), nil
}

func (c *Client) SetLED(ctx context.Context, state bool) error {
	b2i := map[bool]string{true: "1", false: "0"}

	p := url.Values{}
	p.Add("rd_led", b2i[state])
	p.Add("led_cfg", "Apply")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathLedControl+"?"+p.Encode(), nil)
	if err != nil {
		return err
	}

	if _, err := c.client.Do(req); err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	return nil
}

func (c *Client) ClearPortStatistics(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathClearStats+"?clear=Clear", nil)
	if err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	if _, err := c.client.Do(req); err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	return nil
}

func (c *Client) GetPortSpeeds(ctx context.Context) ([]model.PortSpeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.address+PathQoS, nil)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	bodyBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	html := string(bodyBytes)
	maxPorts := parseMaxPorts(html)
	if maxPorts == 0 {
		return nil, clientErrors.ErrInvalidResponse
	}

	speeds := parsePortSpeeds(html, maxPorts)
	if len(speeds) == 0 {
		return nil, clientErrors.ErrInvalidResponse
	}

	portSpeeds := make([]model.PortSpeed, maxPorts)
	for i := range portSpeeds {
		portSpeeds[i] = model.PortSpeed{
			PortNumber:      i + 1,
			IngressRateKbps: speeds[i][0],
			EgressRateKbps:  speeds[i][1],
		}
	}

	return portSpeeds, nil
}

func (c *Client) SetPortSpeeds(ctx context.Context, port, ingress, egress int) error {
	if err := c.validatePort(port); err != nil {
		return err
	}

	if err := c.validateRate(ingress); err != nil {
		return err
	}

	if err := c.validateRate(egress); err != nil {
		return err
	}

	p := url.Values{}
	p.Add("port", strconv.Itoa(port))
	p.Add("ingress_rate", strconv.Itoa(ingress))
	p.Add("egress_rate", strconv.Itoa(egress))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.address+PathQoSSet, strings.NewReader(p.Encode()))
	if err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if _, err := c.client.Do(req); err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	return nil
}

func (c *Client) validatePort(port int) error {
	if port < 1 || port > 24 {
		return errors.Join(errors.New("port must be between 1 and 24"), clientErrors.ErrInvalidRequest)
	}
	return nil
}

func (c *Client) validateRate(rate int) error {
	if rate < 0 || rate > 1000000 {
		return errors.Join(errors.New("rate must be between 0 and 1000000 kbps"), clientErrors.ErrInvalidRequest)
	}
	return nil
}

func (c *Client) Logout(ctx context.Context) error {
	return clientErrors.ErrNotSupported
}

func (c *Client) SetPortState(ctx context.Context, port int, enabled bool) error {
	return clientErrors.ErrNotSupported
}
