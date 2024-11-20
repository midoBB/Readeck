// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forwarded_test

import (
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"testing"

	"codeberg.org/readeck/readeck/pkg/http/forwarded"
	"github.com/stretchr/testify/require"
)

func TestParseXForwardedFor(t *testing.T) {
	tests := []struct {
		header   []string
		expected []net.IP
	}{
		{
			[]string{"192.168.1.1", "10.2.2.1"},
			[]net.IP{net.ParseIP("10.2.2.1"), net.ParseIP("192.168.1.1")},
		},
		{
			[]string{"192.168.1.1, 10.2.2.1"},
			[]net.IP{net.ParseIP("10.2.2.1"), net.ParseIP("192.168.1.1")},
		},
		{
			[]string{" 192.168.1.1,   10.2.2.1  "},
			[]net.IP{net.ParseIP("10.2.2.1"), net.ParseIP("192.168.1.1")},
		},
		{
			[]string{" 192.168.1.1,foo", "bar,10.2.2.1"},
			[]net.IP{net.ParseIP("10.2.2.1"), net.ParseIP("192.168.1.1")},
		},
		{
			[]string{" 192.168.1.1,foo", "bar,10.2.2.1", "fd00:fa::1"},
			[]net.IP{net.ParseIP("fd00:fa::1"), net.ParseIP("10.2.2.1"), net.ParseIP("192.168.1.1")},
		},
		{
			nil, nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			header := http.Header{}
			if test.header != nil {
				header[textproto.CanonicalMIMEHeaderKey("x-forwarded-for")] = test.header
			}

			var res []net.IP
			for _, ip := range forwarded.ParseXForwardedFor(header) {
				res = append(res, ip)
			}

			require.Equal(t, test.expected, res)
		})
	}
}

func TestParseXForwardedProto(t *testing.T) {
	tests := []struct {
		header   []string
		expected string
	}{
		{
			[]string{"https"}, "https",
		},
		{
			[]string{" HTTP "}, "http",
		},
		{
			[]string{"http", "https"}, "http",
		},
		{
			[]string{"https", "foo"}, "https",
		},
		{
			[]string{"foo"}, "",
		},
		{
			nil, "",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			header := http.Header{}
			if test.header != nil {
				header[textproto.CanonicalMIMEHeaderKey("x-forwarded-proto")] = test.header
			}

			res := forwarded.ParseXForwardedProto(header)
			require.Equal(t, test.expected, res)
		})
	}
}

func TestParseXForwardedHost(t *testing.T) {
	tests := []struct {
		header   []string
		expected string
	}{
		{
			[]string{"example.net"}, "example.net",
		},
		{
			[]string{" example.net "}, "example.net",
		},
		{
			nil, "",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			header := http.Header{}
			if test.header != nil {
				header[textproto.CanonicalMIMEHeaderKey("x-forwarded-host")] = test.header
			}

			res := forwarded.ParseXForwardedHost(header)
			require.Equal(t, test.expected, res)
		})
	}
}

func TestParseXRealIP(t *testing.T) {
	tests := []struct {
		header   []string
		expected net.IP
	}{
		{
			[]string{"192.168.1.1"},
			net.ParseIP("192.168.1.1"),
		},
		{
			[]string{"fd00:ab1::2"},
			net.ParseIP("fd00:ab1::2"),
		},
		{
			[]string{"  192.168.1.1   "},
			net.ParseIP("192.168.1.1"),
		},
		{
			[]string{"abc192.168.1.1"},
			nil,
		},
		{
			nil, nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			header := http.Header{}
			if test.header != nil {
				header[textproto.CanonicalMIMEHeaderKey("x-real-ip")] = test.header
			}

			res := forwarded.ParseXRealIP(header)
			require.Equal(t, test.expected, res)
		})
	}
}
