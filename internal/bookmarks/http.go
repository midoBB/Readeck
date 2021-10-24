package bookmarks

import (
	"net/http"
	"net/url"
	"path"

	"github.com/go-chi/chi/v5"

	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/zipfs"
)

// apiRouter is the base bookmark API router.
type apiRouter struct {
	chi.Router
	srv *server.Server
}

type viewsRouter struct {
	chi.Router
	*apiRouter
}

// SetupRoutes mounts the routes for the bookmarks domain.
// "/bm" is a public route outside the api scope in order to avoid
// sending the session cookie.
func SetupRoutes(s *server.Server) {
	// Routes
	// Saved bookmark resources (images & all)
	s.AddRoute("/bm", mediaRoutes(s))

	// API routes
	api := newAPIRouter(s)
	s.AddRoute("/api/bookmarks", api)

	// Website routes
	s.AddRoute("/bookmarks", newViewsRouter(api))
}

// newAPIRouter returns an apiRouter with all the routes set up.
func newAPIRouter(s *server.Server) *apiRouter {
	r := s.AuthenticatedRouter()

	api := &apiRouter{r, s}

	// Bookmark API
	r.With(api.srv.WithPermission("api:bookmarks", "read")).Group(func(r chi.Router) {
		r.With(api.withCollectionFilters, api.withBookmarkList).
			Get("/", api.bookmarkList)
		r.With(api.withBookmark).Route("/{uid:[a-zA-Z0-9]{18,22}}", func(r chi.Router) {
			r.Get("/", api.bookmarkInfo)
			r.Get("/article", api.bookmarkArticle)
			r.Get("/x/*", api.bookmarkResource)
		})

		r.With(api.srv.WithPermission("api:bookmarks", "export")).Group(func(r chi.Router) {
			r.With(
				api.withoutPagination,
				api.withCollectionFilters,
				api.withBookmarkList,
			).Get("/export.{format}", api.bookmarkExport)
			r.With(
				api.withBookmark,
			).Get("/{uid:[a-zA-Z0-9]{18,22}}/article.{format}", api.bookmarkExport)
		})

		r.Route("/labels", func(r chi.Router) {
			r.With(api.withLabelList).Get("/", api.labelList)
			r.With(api.withLabel).Get("/{label}", api.labelInfo)
		})
	})

	r.With(api.srv.WithPermission("api:bookmarks", "write")).Group(func(r chi.Router) {
		r.Post("/", api.bookmarkCreate)
		r.With(api.withBookmark).Group(func(r chi.Router) {
			r.Patch("/{uid:[a-zA-Z0-9]{18,22}}", api.bookmarkUpdate)
			r.Delete("/{uid:[a-zA-Z0-9]{18,22}}", api.bookmarkDelete)
		})
		r.With(api.withLabel).Patch("/labels/{label}", api.labelUpdate)
	})

	// Collection API
	r.Route("/collections", func(r chi.Router) {
		r.With(api.srv.WithPermission("api:bookmarks:collections", "read")).Group(func(r chi.Router) {
			r.With(api.withColletionList).Get("/", api.collectionList)
			r.With(api.withCollection).Get("/{uid:[a-zA-Z0-9]{18,22}}", api.collectionInfo)
		})

		r.With(api.srv.WithPermission("api:bookmarks:collections", "write")).Group(func(r chi.Router) {
			r.Post("/", api.collectionCreate)
			r.With(api.withCollection).Patch("/{uid:[a-zA-Z0-9]{18,22}}", api.collectionUpdate)
			r.With(api.withCollection).Delete("/{uid:[a-zA-Z0-9]{18,22}}", api.collectionDelete)
		})
	})

	return api
}

// newViewsRouter returns an apiRouter with all the routes set up.
func newViewsRouter(api *apiRouter) *viewsRouter {
	r := api.srv.AuthenticatedRouter()

	h := &viewsRouter{r, api}

	// Bookmark and label views
	r.With(h.srv.WithPermission("bookmarks", "read")).Group(func(r chi.Router) {
		r.With(h.withBaseContext, api.withDefaultLimit(24)).Group(func(r chi.Router) {
			r.With(api.withBookmarkList).Get("/", h.bookmarkList)
			r.With(api.withBookmarkFilters, api.withBookmarkList).
				Get("/{filter:(unread|archives|favorites|articles|videos|pictures)}", h.bookmarkList)
			r.With(api.withBookmark).Get("/{uid:[a-zA-Z0-9]{18,22}}", h.bookmarkInfo)
			r.With(api.withLabelList).Get("/labels", h.labelList)
			r.With(api.withLabel, api.withBookmarkList).
				Get("/labels/{label}", h.labelInfo)
		})
	})

	r.With(h.srv.WithPermission("bookmarks", "write")).Group(func(r chi.Router) {
		r.With(h.withBaseContext, api.withDefaultLimit(24)).Group(func(r chi.Router) {
			r.With(api.withBookmarkList).Post("/", h.bookmarkList)
			r.With(api.withBookmark).Group(func(r chi.Router) {
				r.Post("/{uid:[a-zA-Z0-9]{18,22}}", h.bookmarkUpdate)
				r.Post("/{uid:[a-zA-Z0-9]{18,22}}/delete", h.bookmarkDelete)
			})
			r.With(api.withLabel, api.withBookmarkList).
				Post("/labels/{label}", h.labelInfo)
		})
	})

	// Collection views
	r.Route("/collections", func(r chi.Router) {
		r.With(h.srv.WithPermission("bookmarks:collections", "read")).Group(func(r chi.Router) {
			r.With(h.withBaseContext, api.withDefaultLimit(24)).Group(func(r chi.Router) {
				r.With(api.withColletionList).Get("/", h.collectionList)
				r.With(
					api.withCollection,
					api.withCollectionFilters,
					api.withBookmarkList,
				).Get("/{uid:[a-zA-Z0-9]{18,22}}", h.collectionInfo)
			})
		})

		r.With(h.srv.WithPermission("bookmarks:collections", "write")).Group(func(r chi.Router) {
			r.With(h.withBaseContext, api.withDefaultLimit(24)).Group(func(r chi.Router) {
				r.Get("/add", h.collectionCreate)
				r.Post("/add", h.collectionCreate)
				r.With(
					api.withCollection,
					api.withCollectionFilters,
					api.withBookmarkList,
				).Post("/{uid:[a-zA-Z0-9]{18,22}}", h.collectionInfo)
				r.With(
					api.withCollection,
				).Post("/{uid:[a-zA-Z0-9]{18,22}}/delete", h.collectionDelete)
			})
		})
	})

	return h
}

// mediaRoutes serves files from a bookmark's saved archive. It reads
// directly from the zip file and returns the requested file's content.
func mediaRoutes(_ *server.Server) http.Handler {
	r := chi.NewRouter()
	r.Get("/{prefix:[a-zA-Z0-9]{2}}/{fname:[a-zA-Z0-9]+}/{p:^(img|_resources)$}/{name}", func(w http.ResponseWriter, r *http.Request) {
		p := path.Join(
			chi.URLParam(r, "p"),
			chi.URLParam(r, "name"),
		)
		p = path.Clean(p)

		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = p

		zipfile := path.Join(
			StoragePath(),
			chi.URLParam(r, "prefix"),
			chi.URLParam(r, "fname")+".zip",
		)

		fs := zipfs.HTTPZipFile(zipfile)
		fs.ServeHTTP(w, r2)
	})

	return r
}
