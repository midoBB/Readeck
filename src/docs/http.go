// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package docs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/komkom/toml"

	"github.com/readeck/readeck/configs"
	"github.com/readeck/readeck/internal/db"
	"github.com/readeck/readeck/internal/server"
)

type ctxFileKey struct{}
type ctxSectionKey struct{}

type helpHandlers struct {
	chi.Router
	srv *server.Server
}

type licenseInfo struct {
	Name      string
	License   string
	Author    string
	URL       string
	Copyright string
}

const routePrefix = "/docs"

// SetupRoutes mounts the routes for the auth domain.
func SetupRoutes(s *server.Server) {
	handler := &helpHandlers{
		chi.NewRouter(),
		s,
	}

	// File routes
	for _, f := range manifest.Files {
		handler.With(handler.withFile(f)).Get("/"+f.Route, handler.serveStatic)
	}

	// Document routes
	// docHandler serves the document and requires authentication
	docHandler := handler.With(s.AuthenticatedRouter(s.WithRedirectLogin).Middlewares()...)
	for lang, section := range manifest.Sections {
		for _, f := range section.Files {
			// Document
			docHandler.With(
				handler.withFile(f),
				handler.withSection(section),
			).Get("/"+f.Route, handler.serveDocument)

			// Aliases
			for _, alias := range f.Aliases {
				handler.Get("/"+alias, handler.serverRedirect(routePrefix+"/"+f.Route))
			}
		}

		if lang == "en" {
			// About page
			docHandler.With(
				s.WithPermission("system", "read"),
				handler.withSection(section),
			).Get("/about", handler.serverAbout)
		}
	}

	// Main redirection (TODO: do something with user language when we have translations)
	handler.Get("/", handler.serverRedirect(routePrefix+"/en/"))

	// API documentation
	apiSchema := manifest.Files["api.json"]

	docHandler.With(handler.withFile(apiSchema)).Group(func(r chi.Router) {
		r.Get("/api/", handler.serverAPIDocs)
		r.Get("/api.json", handler.serverAPISchema)
	})

	s.AddRoute(routePrefix, handler)
}

func (h *helpHandlers) withFile(f *File) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if f == nil {
				h.srv.Status(w, r, http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), ctxFileKey{}, f)

			h.srv.WriteEtag(w, f)
			h.srv.WithCaching(next).ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *helpHandlers) withSection(section *Section) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxSectionKey{}, section)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *helpHandlers) serveDocument(w http.ResponseWriter, r *http.Request) {
	section, _ := r.Context().Value(ctxSectionKey{}).(*Section)
	f, _ := r.Context().Value(ctxFileKey{}).(*File)

	fd, err := Files.Open(f.File)
	if err != nil {
		h.srv.Error(w, r, err)
		return
	}
	defer fd.Close()

	var contents strings.Builder
	io.Copy(&contents, fd)
	repl := strings.NewReplacer(
		"readeck-instance://", h.srv.AbsoluteURL(r, "/").String(),
	)
	buf := new(bytes.Buffer)
	repl.WriteString(buf, contents.String())

	h.srv.RenderTemplate(w, r, http.StatusOK, "docs/index", server.TC{
		"TOC":   section.TOC,
		"Title": f.Title,
		"HTML":  buf,
	})
}

func (h *helpHandlers) serveStatic(w http.ResponseWriter, r *http.Request) {
	f, _ := r.Context().Value(ctxFileKey{}).(*File)
	fd, err := Files.Open(f.File)
	if err != nil {
		h.srv.Error(w, r, err)
		return
	}
	defer fd.Close()

	http.ServeContent(w, r, f.File, time.Time{}, fd)
}

func (h *helpHandlers) serverRedirect(to string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.srv.Redirect(w, r, to)
	}
}

func (h *helpHandlers) serverAbout(w http.ResponseWriter, r *http.Request) {
	fp, err := assets.Open("licenses/licenses.toml")
	if err != nil {
		h.srv.Error(w, r, err)
		return
	}

	var licenses = map[string][]licenseInfo{}
	dec := json.NewDecoder(toml.New(fp))
	if err = dec.Decode(&licenses); err != nil {
		h.srv.Error(w, r, err)
		return
	}
	slices.SortFunc(licenses["licenses"], func(a, b licenseInfo) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	section, _ := r.Context().Value(ctxSectionKey{}).(*Section)
	h.srv.RenderTemplate(w, r, http.StatusOK, "docs/about", server.TC{
		"TOC":         section.TOC,
		"Version":     configs.Version(),
		"BuildTime":   configs.BuildTime(),
		"Licenses":    licenses["licenses"],
		"OS":          runtime.GOOS,
		"Arch":        runtime.GOARCH,
		"GoVersion":   runtime.Version(),
		"DBConnecter": db.Driver().Name(),
	})
}

func (h *helpHandlers) serverAPISchema(w http.ResponseWriter, r *http.Request) {
	f, _ := r.Context().Value(ctxFileKey{}).(*File)
	fd, err := Files.Open(f.File)
	if err != nil {
		h.srv.Error(w, r, err)
		return
	}
	defer fd.Close()

	var contents strings.Builder
	io.Copy(&contents, fd)
	repl := strings.NewReplacer(
		"__BASE_URI__", h.srv.AbsoluteURL(r, "/api/").String(),
	)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	repl.WriteString(w, contents.String())
}

func (h *helpHandlers) serverAPIDocs(w http.ResponseWriter, r *http.Request) {
	h.srv.RenderTemplate(w, r, http.StatusOK, "docs/api-docs", server.TC{
		"Schema": h.srv.AbsoluteURL(r, "/docs/api.json"),
	})
}
