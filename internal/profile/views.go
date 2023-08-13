// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package profile

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/readeck/readeck/internal/auth"
	"github.com/readeck/readeck/internal/auth/credentials"
	"github.com/readeck/readeck/internal/auth/tokens"
	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/forms"
)

// profileViews is an HTTP handler for the user profile web views
type profileViews struct {
	chi.Router
	*profileAPI
}

const maxCredentials = 20

// newProfileViews returns an new instance of ProfileViews
func newProfileViews(api *profileAPI) *profileViews {
	r := api.srv.AuthenticatedRouter(api.srv.WithRedirectLogin)
	v := &profileViews{r, api}

	r.With(api.srv.WithPermission("profile", "read")).Group(func(r chi.Router) {
		r.Get("/", v.userProfile)
		r.Get("/password", v.userPassword)
	})

	r.With(api.srv.WithPermission("profile", "write")).Group(func(r chi.Router) {
		r.Post("/", v.userProfile)
		r.Post("/password", v.userPassword)
	})

	r.With(api.srv.WithPermission("profile:credentials", "read")).Group(func(r chi.Router) {
		r.With(api.withCredentialList).Get("/credentials", v.credentialList)
		r.With(api.withCredential).Get("/credentials/{uid}", v.credentialInfo)
	})

	r.With(api.srv.WithPermission("profile:credentials", "write")).Group(func(r chi.Router) {
		r.With(api.withCredentialList).Post("/credentials", v.credentialCreate)
		r.With(api.withCredential).Post("/credentials/{uid}", v.credentialInfo)
		r.With(api.withCredential).Post("/credentials/{uid}/delete", v.credentialDelete)
	})

	r.With(api.srv.WithPermission("profile:tokens", "read")).Group(func(r chi.Router) {
		r.With(api.withTokenList).Get("/tokens", v.tokenList)
		r.With(api.withToken).Get("/tokens/{uid}", v.tokenInfo)
	})

	r.With(api.srv.WithPermission("profile:tokens", "write")).Group(func(r chi.Router) {
		r.Post("/tokens", v.tokenCreate)
		r.With(api.withToken).Post("/tokens/{uid}", v.tokenInfo)
		r.With(api.withToken).Post("/tokens/{uid}/delete", v.tokenDelete)
	})

	return v
}

// userProfile handles GET and POST requests on /profile.
func (v *profileViews) userProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.GetRequestUser(r)
	f := newProfileForm()
	f.setUser(user)

	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			if _, err := f.updateUser(user); err != nil {
				v.srv.Log(r).Error(err)
			} else {
				// Set the new seed in the session.
				// We needn't save the session since AddFlash does that already.
				sess := v.srv.GetSession(r)
				sess.Payload.Seed = user.Seed
				v.srv.AddFlash(w, r, "success", "Profile updated")
				v.srv.Redirect(w, r, "profile")
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	ctx := server.TC{
		"Form": f,
	}
	v.srv.RenderTemplate(w, r, 200, "profile/index", ctx)
}

// userPassword handles GET and POST requests on /profile/password
func (v *profileViews) userPassword(w http.ResponseWriter, r *http.Request) {
	f := newPasswordForm()

	if r.Method == http.MethodPost {
		user := auth.GetRequestUser(r)
		f.setUser(user)
		forms.Bind(f, r)
		if f.IsValid() {
			if err := f.updatePassword(user); err != nil {
				v.srv.Log(r).Error(err)
			} else {
				// Set the new seed in the session.
				// We needn't save the session since AddFlash does it already.
				sess := v.srv.GetSession(r)
				sess.Payload.Seed = user.Seed
				v.srv.AddFlash(w, r, "success", "Your password was changed.")
				v.srv.Redirect(w, r, "password")
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	ctx := server.TC{
		"Form": f,
	}
	v.srv.RenderTemplate(w, r, 200, "profile/password", ctx)
}

func (v *profileViews) credentialList(w http.ResponseWriter, r *http.Request) {
	cl := r.Context().Value(ctxCredentialListKey{}).(credentialList)
	ctx := server.TC{
		"Pagination":     cl.Pagination,
		"Credentials":    cl.Items,
		"CanCreate":      cl.Pagination.TotalCount < maxCredentials,
		"MaxCredentials": maxCredentials,
	}
	v.srv.RenderTemplate(w, r, 200, "profile/credential_list", ctx)
}

func (v *profileViews) credentialCreate(w http.ResponseWriter, r *http.Request) {
	cl := r.Context().Value(ctxCredentialListKey{}).(credentialList)
	if cl.Pagination.TotalCount >= maxCredentials {
		v.srv.AddFlash(w, r, "error", "Error: you can not create more credentials")
		v.srv.Redirect(w, r)
		return
	}

	c, passphrase, err := credentials.Credentials.GenerateCredential(auth.GetRequestUser(r).ID)

	if err != nil {
		v.srv.Log(r).WithError(err).Error("server error")
		v.srv.AddFlash(w, r, "error", "An error append while creating your password.")
		v.srv.Redirect(w, r, "credentials")
		return
	}

	v.srv.AddFlash(w, r, "_passphrase", passphrase)
	v.srv.Redirect(w, r, ".", c.UID)
}

func (v *profileViews) credentialInfo(w http.ResponseWriter, r *http.Request) {
	ci := r.Context().Value(ctxCredentialKey{}).(credentialItem)
	f := newCredentialForm(auth.GetRequestUser(r))

	flashes := v.srv.Flashes(r)
	passphrase := ""
	for _, f := range flashes {
		if f.Type == "_passphrase" {
			passphrase = f.Message
		}
	}

	if r.Method == http.MethodGet {
		f.setCredential(ci.Credential)
	}

	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			if err := f.updateCredential(ci.Credential); err != nil {
				v.srv.Log(r).Error(err)
			} else {
				v.srv.AddFlash(w, r, "success", "Password was updated.")
				v.srv.Redirect(w, r, ci.UID)
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	ctx := server.TC{
		"Passphrase": passphrase,
		"Credential": ci,
		"Form":       f,
	}
	v.srv.RenderTemplate(w, r, 200, "profile/credential", ctx)
}

func (v *profileViews) credentialDelete(w http.ResponseWriter, r *http.Request) {
	f := newDeleteCredentialForm()
	f.Get("_to").Set("/profile/credentials")
	forms.Bind(f, r)

	ti := r.Context().Value(ctxCredentialKey{}).(credentialItem)

	f.trigger(ti.Credential)
	v.srv.Redirect(w, r, f.Get("_to").String())
}

func (v *profileViews) tokenList(w http.ResponseWriter, r *http.Request) {
	tl := r.Context().Value(ctxTokenListKey{}).(tokenList)

	ctx := server.TC{
		"Pagination": tl.Pagination,
		"Tokens":     tl.Items,
	}
	v.srv.RenderTemplate(w, r, 200, "profile/token_list", ctx)
}

func (v *profileViews) tokenCreate(w http.ResponseWriter, r *http.Request) {
	t := &tokens.Token{
		UserID:      &auth.GetRequestUser(r).ID,
		IsEnabled:   true,
		Application: "internal",
	}
	if err := tokens.Tokens.Create(t); err != nil {
		v.srv.Log(r).WithError(err).Error("server error")
		v.srv.AddFlash(w, r, "error", "An error append while creating your token.")
		v.srv.Redirect(w, r, "tokens")
		return
	}

	v.srv.AddFlash(w, r, "success", "New token created.")
	v.srv.Redirect(w, r, ".", t.UID)
}

func (v *profileViews) tokenInfo(w http.ResponseWriter, r *http.Request) {
	ti := r.Context().Value(ctxtTokenKey{}).(tokenItem)
	f := newTokenForm(auth.GetRequestUser(r))

	if r.Method == http.MethodGet {
		f.setToken(ti.Token)
	}

	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			if err := f.updateToken(ti.Token); err != nil {
				v.srv.Log(r).Error(err)
			} else {
				v.srv.AddFlash(w, r, "success", "Token was updated.")
				v.srv.Redirect(w, r, ti.UID)
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	jwt, err := tokens.NewJwtToken(ti.UID)
	if err != nil {
		v.srv.Status(w, r, http.StatusInternalServerError)
		return
	}

	ctx := server.TC{
		"Token": ti,
		"JWT":   jwt,
		"Form":  f,
	}

	v.srv.RenderTemplate(w, r, 200, "profile/token", ctx)
}

func (v *profileViews) tokenDelete(w http.ResponseWriter, r *http.Request) {
	f := newDeleteTokenForm()
	f.Get("_to").Set("/profile/tokens")
	forms.Bind(f, r)

	ti := r.Context().Value(ctxtTokenKey{}).(tokenItem)

	f.trigger(ti.Token)
	v.srv.Redirect(w, r, f.Get("_to").String())
}
