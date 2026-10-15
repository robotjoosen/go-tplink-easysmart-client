package gui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
	"github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
)

func Test_Login_PersistsSessionCookie(t *testing.T) {
	var sawCookie string

	mux := http.NewServeMux()
	mux.HandleFunc(PathLogin, func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "Authorization", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc(PathPortStats, func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("Authorization")
		if err == nil {
			sawCookie = cookie.Value
		}
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	err := c.Login(context.Background(), "admin", "admin")
	assert.NoError(t, err)

	_, _ = c.GetPortStats(context.Background()) // page body is irrelevant; only the cookie is checked

	assert.Equal(t, "abc123", sawCookie)
}

func Test_GetCableDiagnostics(t *testing.T) {
	portStatsHTML := `<script>max_port_num = 4;</script>`
	cableDiagHTML := `<script>
var maxPort = 4;
var cablestate = [1, 0, 2, -1, -1];
var cablelength = [10, 0, 5, 0, -1];
</script>`

	var gotQuery url.Values

	mux := http.NewServeMux()
	mux.HandleFunc(PathPortStats, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(portStatsHTML))
	})
	mux.HandleFunc(PathCableDiag, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Write([]byte(cableDiagHTML))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	diagnostics, err := c.GetCableDiagnostics(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, "1", gotQuery.Get("chk_1"))
	assert.Equal(t, "4", gotQuery.Get("chk_4"))
	assert.Equal(t, "Apply", gotQuery.Get("Apply"))
	assert.Empty(t, gotQuery.Get("chk_5"))

	assert.Equal(t, []model.CableDiagnostics{
		{PortNumber: 1, CableState: "Normal", CableStateNumber: 1, Length: 10},
		{PortNumber: 2, CableState: "No Cable", CableStateNumber: 0, Length: 0},
		{PortNumber: 3, CableState: "Open", CableStateNumber: 2, Length: 5},
		{PortNumber: 4, CableState: "Untested", CableStateNumber: -1, Length: 0},
	}, diagnostics)
}

func Test_GetCableDiagnostics_NoMaxPortNum(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(PathPortStats, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>session expired, please log in</body></html>`))
	})
	mux.HandleFunc(PathCableDiag, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(``))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	diagnostics, err := c.GetCableDiagnostics(context.Background())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, clientErrors.ErrInvalidResponse))
	assert.Nil(t, diagnostics)
}

func Test_GetCableDiagnostics_MissingCableStateArrays(t *testing.T) {
	portStatsHTML := `<script>max_port_num = 4;</script>`

	mux := http.NewServeMux()
	mux.HandleFunc(PathPortStats, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(portStatsHTML))
	})
	mux.HandleFunc(PathCableDiag, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<script>var maxPort = 4;</script>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	diagnostics, err := c.GetCableDiagnostics(context.Background())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, clientErrors.ErrInvalidResponse))
	assert.Nil(t, diagnostics)
}

func Test_GetDevice_NonSwitchResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not a switch</html>"))
	}))
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	assert.NotPanics(t, func() {
		_, err := c.GetDevice(context.Background())
		assert.ErrorIs(t, err, clientErrors.ErrInvalidResponse)
	})
}

func Test_Login_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	assert.ErrorIs(t, c.Login(context.Background(), "a", "b"), clientErrors.ErrAuthenticationFailed)
}

func Test_GetDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("\"desc\"\n\"1.2.3.4\"\n\"aa-bb\"\n\"255.0.0.0\"\n\"1.2.3.1\"\n\"fw1\"\n\"hw1\"\n"))
	}))
	defer server.Close()

	c := NewClient(server.Listener.Addr().String())

	d, err := c.GetDevice(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, model.Device{
		Description: "desc", IP: "1.2.3.4", MAC: "aa-bb", SubnetMask: "255.0.0.0",
		DefaultGateway: "1.2.3.1", FirmwareVersion: "fw1", HardwareVersion: "hw1",
	}, d)
}

func portStatsServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
}

func Test_GetPortStats(t *testing.T) {
	server := portStatsServer(`max_port_num = 2;
state:[1,0],
link_status:[1,0],
link_info=new Array("Link Down","100M Full"),
pkts:[1,2,3,4,5,6,7,8]`)
	defer server.Close()

	ports, err := NewClient(server.Listener.Addr().String()).GetPortStats(context.Background())

	assert.NoError(t, err)
	assert.Len(t, ports, 2)
	assert.Equal(t, "100M Full", ports[0].Stats.LinkStatus)
	assert.True(t, ports[0].Stats.Enabled)
	assert.Equal(t, 4, ports[0].Stats.Packets.Receiver.Bad)
	assert.Equal(t, 5, ports[1].Stats.Packets.Transmitter.Good)
}

func Test_GetPortStats_InvalidPages(t *testing.T) {
	pages := map[string]string{
		"login page":        `<html>please log in</html>`,
		"short packets":     "max_port_num = 2;\nstate:[1,1],\nlink_status:[0,0],\nlink_info=new Array(\"a\",\"b\"),\npkts:[1,2,3,4]",
		"link out of range": "max_port_num = 1;\nstate:[1],\nlink_status:[6],\nlink_info=new Array(\"a\",\"b\"),\npkts:[1,2,3,4]",
	}

	for name, page := range pages {
		t.Run(name, func(t *testing.T) {
			server := portStatsServer(page)
			defer server.Close()

			assert.NotPanics(t, func() {
				_, err := NewClient(server.Listener.Addr().String()).GetPortStats(context.Background())
				assert.ErrorIs(t, err, clientErrors.ErrInvalidResponse)
			})
		})
	}
}
