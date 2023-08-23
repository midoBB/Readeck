// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package csp_test

import (
	"net/http"
	"testing"

	"github.com/readeck/readeck/pkg/csp"
	"github.com/stretchr/testify/assert"
)

func TestCSP(t *testing.T) {
	p := csp.Policy{}

	assert.Equal(t, "", p.String())

	p.Add("default-src", csp.None, csp.Self)
	assert.Equal(t, "default-src 'none' 'self'", p.String())

	p.Add("default-src", "example.net")
	assert.Equal(t, "default-src 'none' 'self' example.net", p.String())

	p.Set("default-src", "'self'")
	assert.Equal(t, "default-src 'self'", p.String())

	p.Set("script-src", "'nonce-abcd'", csp.UnsafeInline)
	assert.Equal(t, "default-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'", p.String())

	header := http.Header{}
	p.Write(header)
	assert.Equal(t,
		"default-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'",
		header.Get("Content-Security-Policy"),
	)

	p2 := p.Clone()
	p2.Add("img-src", csp.Self)
	assert.Equal(t,
		"default-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'",
		p.String(),
	)
	assert.Equal(t,
		"default-src 'self'; img-src 'self'; script-src 'nonce-abcd' 'unsafe-inline'",
		p2.String(),
	)

	nonce := csp.MakeNonce()
	assert.Equal(t, 32, len(nonce))
}
