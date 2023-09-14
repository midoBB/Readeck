// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package onboarding

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/server"
	"codeberg.org/readeck/readeck/pkg/forms"
)

// SetupRoutes mounts the routes for the onboarding domain.
func SetupRoutes(s *server.Server) {
	if configs.Config.Commissioned {
		// Do not even add the route if there are users
		return
	}

	r := chi.NewRouter()
	r.Use(
		s.WithSession(),
		s.Csrf,
	)

	h := &viewHandler{r, s}
	s.AddRoute("/onboarding", r)
	r.Get("/", h.onboarding)
	r.Post("/", h.onboarding)
}

type viewHandler struct {
	chi.Router
	srv *server.Server
}

func (h *viewHandler) onboarding(w http.ResponseWriter, r *http.Request) {
	count, err := users.Users.Count()
	if err != nil {
		h.srv.Error(w, r, err)
		return
	}
	if count > 0 {
		// Double check that there's no user yet.
		h.srv.Redirect(w, r, "/login")
		return
	}

	f := newOnboardingForm()

	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			user, err := f.createUser()
			if err != nil {
				h.srv.Log(r).Error(err)
			} else {
				// All good, create a new session for the user
				configs.Config.Commissioned = true

				sess := h.srv.GetSession(r)
				sess.Payload.User = user.ID
				sess.Payload.Seed = user.Seed
				sess.Save(r, w)

				h.srv.Redirect(w, r, "/")
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	ctx := server.TC{
		"Form": f,
	}

	h.srv.RenderTemplate(w, r, http.StatusOK, "auth/onboarding", ctx)
}
