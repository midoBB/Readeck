// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/server"
	"github.com/stretchr/testify/require"
)

func TestInitRequest(t *testing.T) {
	s := server.New("/")
	configs.InitConfiguration()

	tests := []struct {
		RemoteAddr          string
		XForwardedFor       string
		XForwardedHost      string
		XForwardedProto     string
		ExpectedRemoteAddr  string
		ExpectedRemoteHost  string
		ExpectedRemoteProto string
	}{
		{
			"127.0.0.1:1234",
			"203.0.113.1, 192.168.2.1, ::1",
			"example.net",
			"https",
			"203.0.113.1",
			"example.net",
			"https",
		},
		{
			"127.0.0.1:1234",
			"203.0.113.1",
			"example.net:8443",
			"https",
			"203.0.113.1",
			"example.net:8443",
			"https",
		},
		{
			"127.0.0.1:1234",
			"203.0.113.1",
			"example.net",
			"https",
			"203.0.113.1",
			"example.net",
			"https",
		},
		{
			"[::1]:1234",
			"2001:db8:fa::2",
			"example.net",
			"https",
			"2001:db8:fa::2",
			"example.net",
			"https",
		},
		{
			"[fd00::ff01]:1234",
			"203.0.113.1",
			"example.net",
			"https",
			"203.0.113.1",
			"example.net",
			"https",
		},
		{
			"[2001:db8:ff::1]:1234",
			"203.0.113.1",
			"example.net",
			"https",
			"2001:db8:ff::1",
			"test.local",
			"http",
		},
		{
			"128.66.1.1:1234",
			"203.0.113.1",
			"example.net",
			"https",
			"128.66.1.1",
			"test.local",
			"http",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			h := s.InitRequest(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

			r, _ := http.NewRequest("GET", "/", nil)
			r.Host = "test.local"
			r.RemoteAddr = test.RemoteAddr
			r.Header.Set("X-Forwarded-For", test.XForwardedFor)
			r.Header.Set("X-Forwarded-Host", test.XForwardedHost)
			r.Header.Set("X-Forwarded-Proto", test.XForwardedProto)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			assert := require.New(t)
			assert.Equal(test.ExpectedRemoteAddr, r.RemoteAddr)
			assert.Equal(test.ExpectedRemoteHost, r.Host)
			assert.Equal(test.ExpectedRemoteProto, r.URL.Scheme)
			assert.Equal(test.ExpectedRemoteProto+"://"+test.ExpectedRemoteHost+"/", r.URL.String())
		})
	}
}
