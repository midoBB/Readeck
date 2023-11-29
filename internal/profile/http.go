// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package profile provides the user's profile management routes.
package profile

import (
	"codeberg.org/readeck/readeck/internal/server"
)

// SetupRoutes mounts the routes for the auth domain.
func SetupRoutes(s *server.Server) {
	// API routes
	api := newProfileAPI(s)
	s.AddRoute("/api/profile", api)

	// Website views
	s.AddRoute("/profile", newProfileViews(api))
}
