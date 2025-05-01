package gui

import (
	"context"
	"errors"
	"io"
	"net/http"
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
)

type Client struct {
	client  *http.Client
	address string
}

func NewClient(address string) *Client {
	return &Client{
		client:  http.DefaultClient,
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
