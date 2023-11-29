// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package admin contains routes and tasks for Readeck admin.
package admin

import (
	"codeberg.org/readeck/readeck/internal/server"
)

// SetupRoutes mounts the routes for the auth domain.
func SetupRoutes(s *server.Server) {
	// API routes
	api := newAdminAPI(s)

	// API routes
	s.AddRoute("/api/admin", api)

	// Website views
	s.AddRoute("/admin", newAdminViews(api))
}
