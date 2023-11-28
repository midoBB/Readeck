// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"context"
	"net/http"
	"path"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/sessions"
)

type (
	ctxSessionKey struct{}
	ctxFlashKey   struct{}
)

var sessionHandler *sessions.Handler

// InitSession creates the session handler.
func (s *Server) InitSession() error {
	// Create the session handler
	sessionHandler = sessions.NewHandler(
		configs.Config.Server.Session.CookieName,
		configs.CookieHashKey(),
		configs.CookieBlockKey(),
		sessions.Path(path.Join(s.BasePath)),
		sessions.MaxAge(configs.Config.Server.Session.MaxAge),
	)

	return nil
}

// WithSession initialize a session handler that will be available
// on the included routes.
func (s *Server) WithSession() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Store session
			session, _ := sessionHandler.New(r)

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxSessionKey{}, session)

			// Pop messages and store then. We must do it before
			// anything is sent to the client.
			flashes := session.Flashes()
			ctx = context.WithValue(ctx, ctxFlashKey{}, flashes)
			if len(flashes) > 0 {
				session.Save(r, w)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSession returns the session currently stored in context.
// It will panic (on purpose) if the route is not using the
// WithSession() middleware.
func (s *Server) GetSession(r *http.Request) *sessions.Session {
	return r.Context().Value(ctxSessionKey{}).(*sessions.Session)
}

// AddFlash saves a flash message in the session.
func (s *Server) AddFlash(w http.ResponseWriter, r *http.Request, typ, msg string) error {
	session := s.GetSession(r)
	session.AddFlash(typ, msg)
	return session.Save(r, w)
}

// Flashes returns the flash messages retrieved from the session
// in the session middleware.
func (s *Server) Flashes(r *http.Request) []sessions.FlashMessage {
	if msgs := r.Context().Value(ctxFlashKey{}); msgs != nil {
		return msgs.([]sessions.FlashMessage)
	}
	return make([]sessions.FlashMessage, 0)
}
