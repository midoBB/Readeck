// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package dashboard provides routes for Readeck's dashboard.
package dashboard

import (
	"net/http"

	"codeberg.org/readeck/readeck/internal/server"
)

// SetupRoutes mounts the routes for the auth domain.
func SetupRoutes(s *server.Server) {
	s.AddRoute("/", routes(s))
}

func routes(s *server.Server) http.Handler {
	r := s.AuthenticatedRouter(s.WithRedirectLogin)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		s.Redirect(w, r, "/bookmarks/unread")

		// Once we have a real dashboard page, we can restore this
		// s.RenderTemplate(w, r, 200, "dashboard/index.gohtml", server.TC{})
	})

	return r
}
