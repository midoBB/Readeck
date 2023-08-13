// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package cookbook

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type cookbookViews struct {
	chi.Router
	*cookbookAPI
}

func newCookbookViews(api *cookbookAPI) *cookbookViews {
	r := api.srv.AuthenticatedRouter(api.srv.WithRedirectLogin)
	v := &cookbookViews{r, api}

	r.With(api.srv.WithPermission("cookbook", "read")).Group(func(r chi.Router) {
		r.Get("/", v.templateView("prose"))
	})

	return v
}

func (v *cookbookViews) templateView(name string) func(w http.ResponseWriter, r *http.Request) {
	template := fmt.Sprintf("cookbook/%s", name)
	return func(w http.ResponseWriter, r *http.Request) {
		v.srv.RenderTemplate(w, r, 200, template, nil)
	}
}
