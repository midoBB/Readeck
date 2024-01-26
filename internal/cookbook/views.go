// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package cookbook

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"codeberg.org/readeck/readeck/internal/server"
	"codeberg.org/readeck/readeck/pkg/csp"
	"codeberg.org/readeck/readeck/pkg/forms"
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
		r.Get("/colors", v.templateView("colors"))
		r.Get("/forms", v.formsView)
	})

	return v
}

func (v *cookbookViews) templateView(name string) func(w http.ResponseWriter, r *http.Request) {
	template := fmt.Sprintf("cookbook/%s", name)
	return func(w http.ResponseWriter, r *http.Request) {
		policy := server.GetCSPHeader(r).Clone()
		policy.Add("style-src-attr", csp.UnsafeInline)
		policy.Write(w.Header())

		v.srv.RenderTemplate(w, r, 200, template, nil)
	}
}

func (v *cookbookViews) formsView(w http.ResponseWriter, r *http.Request) {
	ctx := server.TC{
		"Form": newCookbookForm(),
	}

	v.srv.RenderTemplate(w, r, 200, "cookbook/forms", ctx)
}

func newCookbookForm() *forms.Form {
	choices := forms.NewListField("choices", func(s string) forms.Field {
		return forms.NewTextField(s)
	}, forms.DefaultListConverter)
	choices.(*forms.ListField).SetChoices(forms.Choices{
		{"a", "Choice A"},
		{"b", "Choice B"},
		{"c", "Choice C"},
	})

	f := forms.Must(
		forms.NewTextField("text"),
		forms.NewChoiceField("select", forms.Choices{
			{"", ""},
			{"choice 1", "Choice 1"},
			{"choice 2", "Choice 2"},
		}),
		choices,
	)

	return f
}
