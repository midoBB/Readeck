package docs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/readeck/readeck/internal/server"
)

type ctxFileKey struct{}
type ctxSectionKey struct{}

type helpHandlers struct {
	chi.Router
	srv *server.Server
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
	for _, section := range manifest.Sections {
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
	}

	// Main redirection (TODO: do something with user language when we have translations)
	handler.Get("/", handler.serverRedirect(routePrefix+"/en/"))

	s.AddRoute(routePrefix, handler)

	// API documentation
	// TODO: this will become available once we have an api-schema.yaml file
	apiSchema := manifest.Files["api-schema.yaml"]
	if apiSchema == nil {
		return
	}

	apiHandler := &helpHandlers{
		s.AuthenticatedRouter(),
		s,
	}
	apiHandler.With(apiHandler.withFile(apiSchema)).Group(func(r chi.Router) {
		r.Get("/", apiHandler.serverAPIDocs)
		r.Get("/schema.yaml", apiHandler.serverAPISchema)
	})
	s.AddRoute("/api/docs", apiHandler)
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

	h.srv.RenderTemplate(w, r, http.StatusOK, "docs/index", server.TC{
		"TOC":   section.TOC,
		"Title": f.Title,
		"HTML":  fd,
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
		"__API_URL__", h.srv.AbsoluteURL(r, "/api/").String(),
	)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	repl.WriteString(w, contents.String())
}

func (h *helpHandlers) serverAPIDocs(w http.ResponseWriter, r *http.Request) {
	h.srv.RenderTemplate(w, r, http.StatusOK, "docs/api-docs", server.TC{
		"Schema": h.srv.AbsoluteURL(r, "/api/docs/schema.yaml"),
	})
}
