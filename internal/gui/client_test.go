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

	_, err = c.GetPortStats(context.Background())
	assert.NoError(t, err)

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
