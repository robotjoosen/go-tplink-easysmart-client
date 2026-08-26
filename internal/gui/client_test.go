package gui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
