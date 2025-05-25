// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"fmt"
	"html"
	"net/http"
	"reflect"
	"strings"

	"github.com/CloudyKit/jet/v6"

	"codeberg.org/readeck/readeck/internal/auth"
	"codeberg.org/readeck/readeck/internal/profile/preferences"
	"codeberg.org/readeck/readeck/internal/templates"
	"codeberg.org/readeck/readeck/pkg/glob"
	"codeberg.org/readeck/readeck/pkg/http/csrf"
	"codeberg.org/readeck/readeck/pkg/libjet"
)

// TC is a simple type to carry template context.
type TC map[string]any

// SetBreadcrumbs sets the current page's breadcrumbs.
func (tc TC) SetBreadcrumbs(items [][2]string) {
	tc["Breadcrumbs"] = items
}

// views holds all the views (templates).
var views *jet.Set

func init() {
	views = templates.Catalog()
}

// GetTemplate returns a template from the current views.
func GetTemplate(name string) (*jet.Template, error) {
	return views.GetTemplate(name)
}

// RenderTemplate yields an HTML response using the given template and context.
func (s *Server) RenderTemplate(w http.ResponseWriter, r *http.Request,
	status int, name string, ctx TC,
) {
	t, err := views.GetTemplate(name)
	if err != nil {
		s.Error(w, r, err)
		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err = t.Execute(w, s.TemplateVars(r), ctx); err != nil {
		panic(err)
	}
}

// RenderTurboStream yields an HTML response with turbo-stream content-type using the
// given template and context. The template result is enclosed in a turbo-stream
// tag with action and target as specified.
// You can call this method as many times as needed to output several turbo-stream tags
// in the same HTTP response.
func (s *Server) RenderTurboStream(
	w http.ResponseWriter, r *http.Request,
	name, action, target string, ctx interface{},
	attrs map[string]string,
) {
	t, err := views.GetTemplate(name)
	if err != nil {
		s.Error(w, r, err)
		return
	}

	extraAttrs := ""
	for k, v := range attrs {
		extraAttrs += k + `="` + html.EscapeString(v) + `" `
	}

	w.Header().Set("Content-Type", "text/vnd.turbo-stream.html; charset=utf-8")

	fmt.Fprintf(w, `<turbo-stream action="%s" %starget="%s"><template>%s`, action, extraAttrs, target, "\n")
	if err = t.Execute(w, s.TemplateVars(r), ctx); err != nil {
		panic(err)
	}
	fmt.Fprint(w, "</template></turbo-stream>\n\n")
}

// initTemplates add global functions to the views.
func (s *Server) initTemplates() {
	views.AddGlobalFunc("assetURL", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("assetURL", 1, 1)
		name := args.Get(0).String()
		r := args.Runtime().Resolve("request").Interface().(*http.Request)

		return reflect.ValueOf(s.AssetURL(r, name))
	})
	views.AddGlobalFunc("urlFor", func(args jet.Arguments) reflect.Value {
		parts := make([]string, args.NumOfArguments())
		for i := 0; i < args.NumOfArguments(); i++ {
			parts[i] = libjet.ToString(args.Get(i))
		}

		r := args.Runtime().Resolve("request").Interface().(*http.Request)
		return reflect.ValueOf(s.AbsoluteURL(r, parts...).EscapedPath())
	})
	views.AddGlobalFunc("pathIs", func(args jet.Arguments) reflect.Value {
		r := args.Runtime().Resolve("request").Interface().(*http.Request)
		cp := "/" + strings.TrimPrefix(r.URL.Path, s.BasePath)
		for i := 0; i < args.NumOfArguments(); i++ {
			if glob.Glob(fmt.Sprintf("%v", args.Get(i)), cp) {
				return reflect.ValueOf(true)
			}
		}
		return reflect.ValueOf(false)
	})
	views.AddGlobalFunc("hasPermission", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("hasPermission", 2, 2)
		obj := libjet.ToString(args.Get(0))
		act := libjet.ToString(args.Get(1))
		r, ok := args.Runtime().Resolve("request").Interface().(*http.Request)
		if !ok {
			return reflect.ValueOf(false)
		}
		return reflect.ValueOf(auth.HasPermission(r, obj, act))
	})
}

// TemplateVars returns the default variables set for a template
// in the request's context.
func (s *Server) TemplateVars(r *http.Request) jet.VarMap {
	cspNonce, _ := r.Context().Value(ctxCSPNonceKey{}).(string)
	tr := s.Locale(r)

	user := auth.GetRequestUser(r)
	session := s.GetSession(r)

	return make(jet.VarMap).
		Set("basePath", s.BasePath).
		Set("csrfName", csrfFieldName).
		Set("csrfToken", csrf.Token(r)).
		Set("currentPath", s.CurrentPath(r)).
		Set("isTurbo", s.IsTurboRequest(r)).
		Set("request", r).
		Set("cspNonce", cspNonce).
		Set("user", user).
		Set("preferences", preferences.New(user, session)).
		Set("flashes", s.Flashes(r)).
		Set("translator", tr).
		Set("gettext", tr.Gettext).
		Set("ngettext", tr.Ngettext).
		Set("pgettext", tr.Pgettext).
		Set("npgettext", tr.Npgettext)
}
