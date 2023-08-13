// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"errors"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/readeck/readeck/internal/acls"
	"github.com/readeck/readeck/internal/auth/credentials"
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
			log.WithError(err).Error("fetching credentials")
		}
		p.denyAccess(w)
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
			log.WithError(err).Error("ACL check error")
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
