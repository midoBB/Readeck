// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package cookbook

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"codeberg.org/readeck/readeck/internal/server"
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
		r.Get("/", v.namedTemplateView("prose"))
		r.Get("/ui", v.uiView)
		r.Get("/{name}", v.templateView)
	})

	return v
}

func (v *cookbookViews) templateView(w http.ResponseWriter, r *http.Request) {
	template := fmt.Sprintf("cookbook/%s", chi.URLParam(r, "name"))
	_, err := server.GetTemplate(template)
	if err != nil {
		v.srv.Log(r).Error("can't load template", slog.Any("err", err))
		v.srv.Status(w, r, http.StatusNotFound)
		return
	}

	v.srv.RenderTemplate(w, r, 200, template, nil)
}

func (v *cookbookViews) namedTemplateView(name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		chi.RouteContext(r.Context()).URLParams.Add("name", name)
		v.templateView(w, r)
	}
}

func (v *cookbookViews) uiView(w http.ResponseWriter, r *http.Request) {
	f := newCookbookForm()
	ef := newCookbookForm()
	forms.UnmarshalValues(ef, url.Values{})

	ctx := server.TC{
		"Form":    f,
		"FormErr": ef,
	}

	v.srv.RenderTemplate(w, r, 200, "cookbook/ui", ctx)
}

func newCookbookForm() *forms.Form {
	choices := forms.NewListField("choices", func(s string) forms.Field {
		return forms.NewTextField(s)
	}, forms.DefaultListConverter, forms.Required)
	choices.(*forms.ListField).SetChoices(forms.Choices{
		{"a", "Choice A"},
		{"b", "Choice B"},
		{"c", "Choice C"},
	})

	f := forms.Must(
		forms.NewTextField("text", forms.Required, forms.IsEmail),
		forms.NewChoiceField("select", forms.Choices{
			{"", ""},
			{"choice 1", "Choice 1"},
			{"choice 2", "Choice 2"},
		}),
		choices,
	)

	return f
}
