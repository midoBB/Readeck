// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package signin

import (
	"net/http"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-chi/chi/v5"

	"github.com/readeck/readeck/internal/auth/tokens"
	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/forms"
)

type authAPI struct {
	chi.Router
	srv *server.Server
}

func newAuthAPI(s *server.Server) *authAPI {
	r := chi.NewRouter()

	api := &authAPI{Router: r, srv: s}
	api.Post("/", api.auth)

	return api
}

// auth performs the user authentication with its username and
// password and then, returns a JWT token tied to this user.
func (api *authAPI) auth(w http.ResponseWriter, r *http.Request) {
	f := newTokenLoginForm()

	forms.Bind(f, r)

	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusBadRequest, f)
		return
	}

	user, err := users.Users.GetOne(goqu.C("username").Eq(f.Get("username").String()))
	if err != nil || !user.CheckPassword(f.Get("password").String()) {
		api.srv.Message(w, r, &server.Message{
			Status:  http.StatusForbidden,
			Message: "Invalid user and/or password",
		})
		return
	}

	t := &tokens.Token{
		UserID:      &user.ID,
		IsEnabled:   true,
		Application: f.Get("application").String(),
	}
	if err := tokens.Tokens.Create(t); err != nil {
		api.srv.Error(w, r, err)
		return
	}

	token, err := tokens.NewJwtToken(t.UID)
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	api.srv.Render(w, r, http.StatusCreated, tokenReturn{
		UID:   t.UID,
		Token: token.String(),
	})
}

type tokenReturn struct {
	UID   string `json:"id"`
	Token string `json:"token"`
}
