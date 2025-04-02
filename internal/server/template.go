// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"image/color"
	"io"
	"io/fs"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/skip2/go-qrcode"

	"codeberg.org/readeck/readeck/assets"
	"codeberg.org/readeck/readeck/internal/auth"
	"codeberg.org/readeck/readeck/internal/email"
	"codeberg.org/readeck/readeck/internal/profile/preferences"
	"codeberg.org/readeck/readeck/locales"
	"codeberg.org/readeck/readeck/pkg/csrf"
	"codeberg.org/readeck/readeck/pkg/glob"
	"codeberg.org/readeck/readeck/pkg/libjet"
	"codeberg.org/readeck/readeck/pkg/strftime"
)

// TC is a simple type to carry template context.
type TC map[string]interface{}

// SetBreadcrumbs sets the current page's breadcrumbs.
func (tc TC) SetBreadcrumbs(items [][2]string) {
	tc["Breadcrumbs"] = items
}

// tplLoader implements a jet.Loader using fs.FS so we can use it
// with embed fs.
type tplLoader struct {
	fs.FS
}

// Exists returns true if the template exists in the filesystem.
func (l *tplLoader) Exists(templatePath string) bool {
	_, err := l.Open(templatePath)
	return err == nil && !os.IsNotExist(err)
}

// Open opens the template at the give path.
func (l *tplLoader) Open(templatePath string) (io.ReadCloser, error) {
	templatePath = strings.TrimPrefix(templatePath, "/")
	return l.FS.Open(templatePath)
}

// views holds all the views (templates).
var views *jet.Set

func init() {
	loader := &tplLoader{assets.TemplatesFS()}
	views = jet.NewSet(
		loader,
		jet.WithTemplateNameExtensions([]string{"", ".jet.html"}),
	)
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
	for k, v := range libjet.FuncMap() {
		views.AddGlobalFunc(k, v)
	}

	for k, v := range libjet.VarMap() {
		views.AddGlobal(k, v)
	}

	views.AddGlobalFunc("date", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("date", 2, 2)
		v, isNil := libjet.Indirect(args.Get(0))
		if isNil {
			return reflect.ValueOf("")
		}

		date, ok := v.(time.Time)
		if !ok {
			panic("first argument must be a time.Time value or pointer")
		}

		var result string
		tr, ok := args.Runtime().Resolve("translator").Interface().(*locales.Locale)
		if !ok {
			result = strftime.Strftime(libjet.ToString(args.Get(1)), date)
		} else {
			result = strftime.New(tr).Strftime(libjet.ToString(args.Get(1)), date)
		}

		return reflect.ValueOf(result)
	})

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
	views.AddGlobalFunc("qrcode", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("qrcode", 1, 3)
		value := args.Get(0).String()
		size := 240
		clr := "#000000"
		if args.NumOfArguments() > 1 {
			size = libjet.ToInt[int](args.Get(1))
		}

		if args.NumOfArguments() > 2 {
			clr = libjet.ToString(args.Get(2))
		}

		qr, err := qrcode.New(value, qrcode.Medium)
		if err != nil {
			panic(err)
		}
		qr.ForegroundColor, _ = parseHexColor(clr)
		qr.DisableBorder = true
		buf, err := qr.PNG(size)
		if err != nil {
			panic(err)
		}

		return reflect.ValueOf("data:image/png;base64," + base64.StdEncoding.EncodeToString(buf))
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
		Set("canSendEmail", email.CanSendEmail()).
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

func parseHexColor(s string) (c color.RGBA, err error) {
	c.A = 255
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%01x%01x%01x", &c.R, &c.G, &c.B)
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = errors.New("invalid length")
	}
	return
}
