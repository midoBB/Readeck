// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/readeck/readeck/configs"
	"github.com/readeck/readeck/pkg/csp"
)

type (
	ctxCSPNonceKey     struct{}
	ctxCSPKey          struct{}
	unauthorizedCtxKey struct{}
)

const (
	unauthorizedDefault = iota
	unauthorizedRedir
)

func setHost(r *http.Request) error {
	xfh := r.Header.Get("X-Forwarded-Host")
	if xfh == "" {
		return nil
	}
	pair := strings.SplitN(xfh, ":", 2)
	host := pair[0]

	if len(pair) > 1 {
		port, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return err
		}

		r.Host = fmt.Sprintf("%s:%d", host, port)

	} else {
		r.Host = host
	}

	return nil
}

func checkHost(r *http.Request) error {
	host := r.Host
	port := r.URL.Port()
	if port != "" {
		host = strings.TrimSuffix(host, ":"+port)
	}
	host = strings.TrimSuffix(host, ".")

	for _, x := range configs.Config.Server.AllowedHosts {
		if x == host {
			return nil
		}
	}
	return fmt.Errorf("host is not allowed: %s", host)
}

func setProto(r *http.Request) error {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		return nil
	}
	if proto != "http" && proto != "https" {
		return fmt.Errorf("invalid x-forwarded-proto %s", proto)
	}
	r.URL.Scheme = proto
	return nil
}

// InitRequest update the scheme and host on the incoming
// HTTP request URL (r.URL), based on provided headers and/or
// current environnement.
//
// It also checks the validity of the host header when the server
// is not running in dev mode.
func (s *Server) InitRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set host
		if configs.Config.Server.UseXForwardedHost {
			if err := setHost(r); err != nil {
				s.Log(r).WithError(err).Error("server error")
				s.Status(w, r, http.StatusBadRequest)
				return
			}
		}
		r.URL.Host = r.Host

		// Check host
		if !configs.Config.Main.DevMode {
			if err := checkHost(r); err != nil {
				s.Log(r).WithError(err).Error("server error")
				s.Status(w, r, http.StatusBadRequest)
				return
			}
		}

		// Set scheme
		r.URL.Scheme = "http"
		if configs.Config.Server.UseXForwardedProto {
			if err := setProto(r); err != nil {
				s.Log(r).WithError(err).Error("server error")
				s.Status(w, r, http.StatusBadRequest)
				return
			}
		} else if r.TLS != nil {
			r.URL.Scheme = "https"
		}

		next.ServeHTTP(w, r)
	})
}

// getDefaultCSP returns the default Content Security Policy
// There are no definition on script-src and style-src because
// the SetSecurityHeaders middleware will set a nonce value
// for each of them.
func getDefaultCSP() csp.Policy {
	return csp.Policy{
		"base-uri":        {csp.None},
		"default-src":     {csp.Self},
		"font-src":        {csp.Self},
		"form-action":     {csp.Self},
		"frame-ancestors": {csp.None},
		"img-src":         {csp.Self, csp.Data},
		"media-src":       {csp.Self, csp.Data},
		"object-src":      {csp.None},
		"script-src":      {},
		"style-src":       {},
	}
}

// GetCSPHeader extracts the current CSPHeader from the request's context.
func GetCSPHeader(r *http.Request) csp.Policy {
	if c, ok := r.Context().Value(ctxCSPKey{}).(csp.Policy); ok {
		return c
	}
	return getDefaultCSP()
}

// SetSecurityHeaders adds some headers to improve client side security.
func (s *Server) SetSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := csp.MakeNonce()

		policy := getDefaultCSP()
		policy.Add("script-src", fmt.Sprintf("'nonce-%s'", nonce), csp.UnsafeInline)
		policy.Add("style-src", fmt.Sprintf("'nonce-%s'", nonce), csp.UnsafeInline)

		policy.Write(w.Header())
		w.Header().Set("Permissions-Policy", "interest-cohort=()")
		w.Header().Set("Referrer-Policy", "same-origin, strict-origin")
		w.Header().Add("X-Frame-Options", "DENY")
		w.Header().Add("X-Content-Type-Options", "nosniff")
		w.Header().Add("X-XSS-Protection", "1; mode=block")

		ctx := context.WithValue(r.Context(), ctxCSPNonceKey{}, nonce)
		ctx = context.WithValue(ctx, ctxCSPKey{}, policy)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// unauthorizedHandler is a handler used by the session authentication provider.
// It sends different responses based on the context.
func (s *Server) unauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	unauthorizedCtx, _ := r.Context().Value(unauthorizedCtxKey{}).(int)

	switch unauthorizedCtx {
	case unauthorizedDefault:
		w.Header().Add("WWW-Authenticate", `Basic realm="Readeck Authentication"`)
		w.Header().Add("WWW-Authenticate", `Bearer realm="Bearer token"`)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "Unauthorized")
	case unauthorizedRedir:
		redir := s.AbsoluteURL(r, "/login")

		// Add the current path as a redirect query parameter
		// to the login route
		q := redir.Query()
		q.Add("r", s.CurrentPath(r))
		redir.RawQuery = q.Encode()

		w.Header().Set("Location", redir.String())
		w.WriteHeader(http.StatusSeeOther)
	}
}

// WithRedirectLogin sets the unauthorized handler to redirect to the login page.
func (s *Server) WithRedirectLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), unauthorizedCtxKey{}, unauthorizedRedir)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
