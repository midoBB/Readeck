// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/pkg/csp"
	"codeberg.org/readeck/readeck/pkg/http/forwarded"
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
	xfh := forwarded.ParseXForwardedHost(r.Header)
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
	// If allowed_hosts is not set, do not check the hostname.
	if len(configs.Config.Server.AllowedHosts) == 0 {
		return nil
	}

	host := r.Host
	port := r.URL.Port()
	if port != "" {
		host = strings.TrimSuffix(host, ":"+port)
	}
	host = strings.TrimSuffix(host, ".")

	if slices.Contains(configs.Config.Server.AllowedHosts, host) {
		return nil
	}
	return fmt.Errorf("host is not allowed: %s", host)
}

func setProto(r *http.Request) {
	proto := forwarded.ParseXForwardedProto(r.Header)
	if proto != "" {
		r.URL.Scheme = proto
	}
}

func setIP(r *http.Request, knownProxies []*net.IPNet) {
	// The IPs or IP ranges of the trusted reverse proxies are configured.
	// The X-Forwarded-For IP list is searched from the rightmost, skipping all addresses that are
	// on the trusted proxy list. The first non-matching address is the target address.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For
	for _, ip := range forwarded.ParseXForwardedFor(r.Header) {
		if slices.ContainsFunc(knownProxies, func(x *net.IPNet) bool {
			return x.Contains(ip)
		}) {
			continue
		}
		r.RemoteAddr = ip.String()
		break
	}
}

// InitRequest update the scheme and host on the incoming
// HTTP request URL (r.URL), based on provided headers and/or
// current environnement.
//
// It also checks the validity of the host header when the server
// is not running in dev mode.
func (s *Server) InitRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First, always remove the port from RenoteAddr
		r.RemoteAddr, _, _ = net.SplitHostPort(r.RemoteAddr)
		remoteIP := net.ParseIP(r.RemoteAddr)
		trusted := slices.ContainsFunc(configs.TrustedProxies(), func(ip *net.IPNet) bool {
			return ip.Contains(remoteIP)
		})

		// Set host
		if trusted {
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
		if trusted {
			setProto(r)
		} else if r.TLS != nil {
			r.URL.Scheme = "https"
		}

		// Set real IP
		if trusted {
			setIP(r, configs.TrustedProxies())
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
		"script-src":      {csp.ReportSample},
		"style-src":       {csp.ReportSample},
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
		var nonce string
		if nonce = r.Header.Get("x-turbo-nonce"); nonce == "" {
			nonce = csp.MakeNonce()
		}

		policy := getDefaultCSP()
		policy.Add("script-src", fmt.Sprintf("'nonce-%s'", nonce), csp.UnsafeInline)
		policy.Add("style-src", fmt.Sprintf("'nonce-%s'", nonce), csp.UnsafeInline)
		policy.Add("report-uri", s.AbsoluteURL(r, "/logger/csp-report").String())

		policy.Write(w.Header())
		w.Header().Set("Permissions-Policy", "interest-cohort=()")
		w.Header().Set("Referrer-Policy", "same-origin, strict-origin")
		w.Header().Add("X-Frame-Options", "DENY")
		w.Header().Add("X-Content-Type-Options", "nosniff")
		w.Header().Add("X-XSS-Protection", "1; mode=block")
		w.Header().Add("X-Robots-Tag", "noindex, nofollow, noarchive")

		ctx := context.WithValue(r.Context(), ctxCSPNonceKey{}, nonce)
		ctx = context.WithValue(ctx, ctxCSPKey{}, policy)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) cspReport(w http.ResponseWriter, r *http.Request) {
	report := map[string]any{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&report); err != nil {
		s.Log(r).WithError(err).Error("server error")
		return
	}

	s.Log(r).WithField("report", report["csp-report"]).Warn("CSP violation")
	w.WriteHeader(http.StatusNoContent)
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
		if !configs.Config.Commissioned {
			s.Redirect(w, r, "/onboarding")
			return
		}

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
