// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package cookbook provides routes for testing and design previews.
// The routes are only available to admin users.
package cookbook

import (
	"codeberg.org/readeck/readeck/internal/server"
)

// SetupRoutes mounts the routes for the cookbook domain.
func SetupRoutes(s *server.Server) {
	// API routes
	api := newCookbookAPI(s)
	s.AddRoute("/api/cookbook", api)

	// Views
	s.AddRoute("/cookbook", newCookbookViews(api))
}
