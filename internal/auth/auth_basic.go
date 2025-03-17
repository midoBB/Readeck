// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/internal/acls"
	"codeberg.org/readeck/readeck/internal/auth/credentials"
)

// BasicAuthProvider handles basic HTTP authentication method
// with "Authorization: Basic {payload}" header.
type BasicAuthProvider struct{}

// IsActive returns true when the client submits basic HTTP authorization
// header.
func (p *BasicAuthProvider) IsActive(r *http.Request) bool {
	_, _, ok := r.BasicAuth()
	return ok
}

// Authenticate performs the authentication using the HTTP basic authentication
// information provided.
func (p *BasicAuthProvider) Authenticate(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		p.denyAccess(w)
		return r, errors.New("invalid authentication header")
	}

	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		p.denyAccess(w)
		return r, errors.New("no username and/or password provided")
	}

	uc, err := credentials.Credentials.GetUser(username, password)
	if err != nil {
		if err != credentials.ErrNotFound {
			slog.Error("fetching credentials", slog.Any("err", err))
		}
		p.denyAccess(w)
		return r, err
	}

	if err := uc.Credential.Update(goqu.Record{
		"last_used": time.Now().UTC(),
	}); err != nil {
		return r, err
	}

	return SetRequestAuthInfo(r, &Info{
		Provider: &ProviderInfo{
			Name:        "basic auth",
			Application: uc.Credential.Name,
			ID:          uc.Credential.UID,
			Roles:       uc.Credential.Roles,
		},
		User: uc.User,
	}), nil
}

// HasPermission checks the permission on the current authentication provider role
// list. If the role list is empty, the user permissions apply.
func (p *BasicAuthProvider) HasPermission(r *http.Request, obj, act string) bool {
	if len(GetRequestAuthInfo(r).Provider.Roles) == 0 {
		return true
	}

	for _, scope := range GetRequestAuthInfo(r).Provider.Roles {
		if ok, err := acls.Check(scope, obj, act); err != nil {
			slog.Error("ACL check error", slog.Any("err", err))
		} else if ok {
			return true
		}
	}

	return false
}

// GetPermissions returns all the permissions attached to the current authentication provider
// role list. If no role is defined, it will fallback to the user permission list.
func (p *BasicAuthProvider) GetPermissions(r *http.Request) []string {
	if len(GetRequestAuthInfo(r).Provider.Roles) == 0 {
		return nil
	}

	plist, _ := acls.GetPermissions(GetRequestAuthInfo(r).Provider.Roles...)
	return plist
}

// CsrfExempt is always true for this provider.
func (p *BasicAuthProvider) CsrfExempt(_ *http.Request) bool {
	return true
}

func (p *BasicAuthProvider) denyAccess(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="Restricted"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
}
