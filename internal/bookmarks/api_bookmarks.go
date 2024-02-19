// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/go-chi/chi/v5"
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"

	"codeberg.org/readeck/readeck/internal/auth"
	"codeberg.org/readeck/readeck/internal/db/filters"
	"codeberg.org/readeck/readeck/internal/server"
	"codeberg.org/readeck/readeck/pkg/annotate"
	"codeberg.org/readeck/readeck/pkg/forms"
	"codeberg.org/readeck/readeck/pkg/zipfs"
)

type (
	ctxAnnotationListKey    struct{}
	ctxBookmarkKey          struct{}
	ctxBookmarkListKey      struct{}
	ctxBookmarkListTagerKey struct{}
	ctxLabelKey             struct{}
	ctxLabelListKey         struct{}
	ctxSharedInfoKey        struct{}
	ctxFiltersKey           struct{}
	ctxDefaultLimitKey      struct{}
)

var sharingDuration = 24 * time.Hour

// bookmarkList renders a paginated list of the connected
// user bookmarks in JSON.
func (api *apiRouter) bookmarkList(w http.ResponseWriter, r *http.Request) {
	bl := r.Context().Value(ctxBookmarkListKey{}).(bookmarkList)

	bl.Items = make([]bookmarkItem, len(bl.items))
	for i, item := range bl.items {
		bl.Items[i] = newBookmarkItem(api.srv, r, item, ".")
	}

	api.srv.SendPaginationHeaders(w, r, bl.Pagination)
	api.srv.Render(w, r, http.StatusOK, bl.Items)
}

// bookmarkInfo renders a given bookmark items in JSON.
func (api *apiRouter) bookmarkInfo(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	item := newBookmarkItem(api.srv, r, b, "./..")
	item.Errors = b.Errors
	if err := item.setEmbed(); err != nil {
		api.srv.Log(r).Error(err)
	}

	if api.srv.IsTurboRequest(r) {
		api.srv.RenderTurboStream(w, r,
			"/bookmarks/components/card", "replace",
			"bookmark-card-"+b.UID, item)
		return
	}

	api.srv.Render(w, r, http.StatusOK, item)
}

// bookmarkArticle renders the article HTML content of a bookmark.
// Note that only the body's content is rendered.
func (api *apiRouter) bookmarkArticle(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)

	bi := newBookmarkItem(api.srv, r, b, "")
	buf, err := bi.getArticle()
	if err != nil {
		api.srv.Log(r).Error(err)
	}

	if api.srv.IsTurboRequest(r) {
		api.srv.RenderTurboStream(w, r,
			"/bookmarks/components/content_block", "replace",
			"bookmark-content-"+b.UID, map[string]interface{}{
				"Item": bi,
				"HTML": buf,
				"Out":  w,
			})
		api.srv.RenderTurboStream(w, r,
			"/bookmarks/components/sidebar", "replace",
			"bookmark-sidebar-"+b.UID, map[string]interface{}{
				"Item": bi,
			},
		)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	io.Copy(w, buf)
}

// bookmarkExport renders a list of bookmarks in the requested export format.
func (api *apiRouter) bookmarkExport(w http.ResponseWriter, r *http.Request) {
	var fn func(http.ResponseWriter, *http.Request, ...*Bookmark)

	// Check if we have a valid format
	format := chi.URLParam(r, "format")
	switch format {
	case "epub":
		fn = api.exportBookmarksEPUB
	case "md":
		fn = api.exportBookmarksMD
	default:
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}

	// If we have a bookmark list
	bl, ok := r.Context().Value(ctxBookmarkListKey{}).(bookmarkList)
	if ok {
		fn(w, r, bl.items...)
		return
	}

	// Just one bookmark?
	b, ok := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	if ok {
		fn(w, r, b)
		return
	}

	api.srv.Status(w, r, http.StatusNotFound)
}

// bookmarkCreate creates a new bookmark.
func (api *apiRouter) bookmarkCreate(w http.ResponseWriter, r *http.Request) {
	var err error
	ct, _, _ := mime.ParseMediaType(r.Header.Get("content-type"))

	f := newCreateForm(auth.GetRequestUser(r).ID, api.srv.GetReqID(r))

	if ct == "multipart/form-data" {
		// A multipart form must provide a section with the url and others "resource"
		// with each cached resource.
		f.Bind()
		err = f.loadMultipart(r)
		if err != nil {
			f.AddErrors("", fmt.Errorf("Unable to process input data"))
			api.srv.Log(r).WithError(err).Error("input error")
		}
	} else {
		forms.Bind(f, r)
	}

	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusUnprocessableEntity, f)
		return
	}

	b, err := f.createBookmark()
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	w.Header().Add(
		"Location",
		api.srv.AbsoluteURL(r, ".", b.UID).String(),
	)
	w.Header().Add("bookmark-id", b.UID)
	server.NewLink(api.srv.AbsoluteURL(r, "/bookmarks", b.UID).String()).
		WithRel("alternate").
		WithType("text/html").
		Write(w)

	api.srv.TextMessage(w, r, http.StatusAccepted, "Link submited")
}

// bookmarkUpdate updates an existing bookmark.
func (api *apiRouter) bookmarkUpdate(w http.ResponseWriter, r *http.Request) {
	f := newUpdateForm()
	forms.Bind(f, r)

	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusBadRequest, f)
		return
	}

	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)

	updated, err := f.update(b)
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	updated["href"] = api.srv.AbsoluteURL(r).String()

	// On a turbo request, we'll return the updated components.
	if api.srv.IsTurboRequest(r) {
		item := newBookmarkItem(api.srv, r, b, "./..")

		_, withTitle := updated["title"]
		_, withLabels := updated["labels"]
		_, withMarked := updated["is_marked"]
		_, withArchived := updated["is_archived"]
		_, withDeleted := updated["is_deleted"]

		if withTitle {
			api.srv.RenderTurboStream(w, r,
				"/bookmarks/components/title_form", "replace",
				"bookmark-title-"+b.UID, item)
		}
		if withLabels {
			api.srv.RenderTurboStream(w, r,
				"/bookmarks/components/labels", "replace",
				"bookmark-label-list-"+b.UID, item)
		}
		if withMarked || withArchived || withDeleted {
			api.srv.RenderTurboStream(w, r,
				"/bookmarks/components/actions", "replace",
				"bookmark-actions-"+b.UID, item)
			api.srv.RenderTurboStream(w, r,
				"/bookmarks/components/card", "replace",
				"bookmark-card-"+b.UID, item)
		}
		if withMarked || withArchived {
			api.srv.RenderTurboStream(w, r,
				"/bookmarks/components/bottom_actions", "replace",
				"bookmark-bottom-actions-"+b.UID, item)
		}
		return
	}

	w.Header().Add(
		"Location",
		updated["href"].(string),
	)
	api.srv.Render(w, r, http.StatusOK, updated)
}

// bookmarkDelete deletes a bookmark.
func (api *apiRouter) bookmarkDelete(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)

	if err := b.Update(map[string]interface{}{}); err != nil {
		api.srv.Error(w, r, err)
		return
	}

	f := newDeleteForm()
	f.Get("cancel").Set(false)
	if err := f.trigger(b); err != nil {
		api.srv.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *apiRouter) bookmarkShare(w http.ResponseWriter, r *http.Request) {
	info := r.Context().Value(ctxSharedInfoKey{}).(sharedBookmarkItem)

	if api.srv.IsTurboRequest(r) {
		api.srv.RenderTurboStream(w, r,
			"/bookmarks/components/public_share", "replace",
			"bookmark-share-"+info.ID, info)
		return
	}

	api.srv.Render(w, r, http.StatusCreated, info)
}

// bookmarkResource is the route returning any resource
// from a given bookmark. The resource is extracted from
// the sidecar zip file of a bookmark.
// Note that for images, we'll use another route that is not
// authenticated and thus, much faster.
func (api *apiRouter) bookmarkResource(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	p := path.Clean(chi.URLParam(r, "*"))

	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = p

	fs := zipfs.HTTPZipFile(b.getFilePath())
	fs.ServeHTTP(w, r2)
}

// labelList returns the list of all labels.
func (api *apiRouter) labelList(w http.ResponseWriter, r *http.Request) {
	res := r.Context().Value(ctxLabelListKey{}).([]*labelItem)
	for i, item := range res {
		u := api.srv.AbsoluteURL(r, ".")
		u.Path += item.Name.Path()
		res[i].Href = u.String()
	}

	api.srv.Render(w, r, http.StatusOK, res)
}

// labelInfo return the information about a label.
func (api *apiRouter) labelInfo(w http.ResponseWriter, r *http.Request) {
	label := r.Context().Value(ctxLabelKey{}).(string)
	ds := Bookmarks.Query().
		Select("id").
		Where(
			goqu.C("user_id").Table("b").Eq(auth.GetRequestUser(r).ID),
		)
	ds = filters.JSONListFilter(ds, goqu.I("b.labels").Eq(label))
	count, err := ds.Count()
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	if count == 0 {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}

	u := api.srv.AbsoluteURL(r, "/api/bookmarks")
	q := u.Query()
	q.Add("label", fmt.Sprintf(`"%s"`, label))
	u.RawQuery = q.Encode()

	api.srv.Render(w, r, http.StatusOK, map[string]interface{}{
		"name":           label,
		"count":          count,
		"href":           api.srv.AbsoluteURL(r).String(),
		"href_bookmarks": u.String(),
	})
}

func (api *apiRouter) labelUpdate(w http.ResponseWriter, r *http.Request) {
	label := r.Context().Value(ctxLabelKey{}).(string)
	f := newLabelForm()
	forms.Bind(f, r)

	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusBadRequest, f)
		return
	}

	ids, err := Bookmarks.RenameLabel(auth.GetRequestUser(r), label, f.Get("name").String())
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}
	if len(ids) == 0 {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}
}

func (api *apiRouter) labelDelete(w http.ResponseWriter, r *http.Request) {
	label := r.Context().Value(ctxLabelKey{}).(string)

	ids, err := Bookmarks.RenameLabel(auth.GetRequestUser(r), label, "")
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}
	if len(ids) == 0 {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *apiRouter) bookmarkAnnotations(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	if b.Annotations != nil {
		api.srv.Render(w, r, http.StatusOK, b.Annotations)
		return
	}

	api.srv.Render(w, r, http.StatusOK, BookmarkAnnotations{})
}

func (api *apiRouter) annotationCreate(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	f := newAnnotationForm()
	forms.Bind(f, r)
	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusUnprocessableEntity, f)
		return
	}

	bi := newBookmarkItem(api.srv, r, b, "")
	annotation, err := f.addToBookmark(&bi)
	if err != nil {
		if errors.As(err, &annotate.ErrAnotate) {
			api.srv.Message(w, r, &server.Message{
				Status:  http.StatusBadRequest,
				Message: err.Error(),
			})
		} else {
			api.srv.Error(w, r, err)
		}
		return
	}

	w.Header().Add("Location", api.srv.AbsoluteURL(r, ".", annotation.ID).String())
	api.srv.Render(w, r, 200, annotation)
}

func (api *apiRouter) annotationDelete(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	id := chi.URLParam(r, "id")
	if b.Annotations == nil {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}
	if b.Annotations.get(id) == nil {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}

	b.Annotations.delete(id)
	err := b.Update(map[string]interface{}{
		"annotations": b.Annotations,
	})
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (api *apiRouter) annotationList(w http.ResponseWriter, r *http.Request) {
	al := r.Context().Value(ctxAnnotationListKey{}).(annotationList)

	api.srv.SendPaginationHeaders(w, r, al.Pagination)
	api.srv.Render(w, r, 200, al.Items)
}

// withBookmark returns a router that will fetch a bookmark and add it into the
// request's context. It also deals with if-modified-since header.
func (api *apiRouter) withBookmark(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := chi.URLParam(r, "uid")

		b, err := Bookmarks.GetOne(
			goqu.C("uid").Eq(uid),
			goqu.C("user_id").Eq(auth.GetRequestUser(r).ID),
		)
		if err != nil {
			api.srv.Status(w, r, http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), ctxBookmarkKey{}, b)

		if b.State == StateLoaded {
			api.srv.WriteLastModified(w, r, b, auth.GetRequestUser(r))
			api.srv.WriteEtag(w, r, b, auth.GetRequestUser(r))
		}

		w.Header().Add("bookmark-id", b.UID)
		server.NewLink(api.srv.AbsoluteURL(r, "/bookmarks", b.UID).String()).
			WithRel("alternate").
			WithType("text/html").
			Write(w)

		api.srv.WithCaching(next).ServeHTTP(w, r.WithContext(ctx))
	})
}

func (api *apiRouter) withBookmarkFilters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter := chi.URLParam(r, "filter")
		filters := newFilterForm()

		switch filter {
		case "unread":
			filters.setArchived(false)
		case "archives":
			filters.setArchived(true)
		case "favorites":
			filters.setMarked(true)
		case "articles":
			filters.setType("article")
		case "pictures":
			filters.setType("photo")
		case "videos":
			filters.setType("video")
		}

		next.ServeHTTP(w, r.Clone(filters.saveContext(r.Context())))
	})
}

func (api *apiRouter) withLabel(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		label, err := url.QueryUnescape(chi.URLParam(r, "label"))
		if err != nil {
			api.srv.Error(w, r, err)
			return
		}

		ctx := context.WithValue(r.Context(), ctxLabelKey{}, label)

		filters := newFilterForm()
		filters.Get("labels").Set(fmt.Sprintf(`"%s"`, label))
		ctx = filters.saveContext(ctx)

		next.ServeHTTP(w, r.Clone(ctx))
	})
}

func (api *apiRouter) withDefaultLimit(limit int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxDefaultLimitKey{}, limit)
			next.ServeHTTP(w, r.Clone(ctx))
		})
	}
}

func (api *apiRouter) withoutPagination(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := newContextFilterForm(r.Context())
		f.noPagination = true
		next.ServeHTTP(w, r.Clone(f.saveContext(r.Context())))
	})
}

func (api *apiRouter) withCollectionFilters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var c *Collection
		var ok bool
		var err error
		ctx := r.Context()
		c, ok = ctx.Value(ctxCollectionKey{}).(*Collection)
		if !ok {
			// No collection in context, let's see if we have an ID
			uid := r.URL.Query().Get("collection")
			if uid == "" {
				next.ServeHTTP(w, r)
				return
			}

			c, err = Collections.GetOne(
				goqu.C("uid").Eq(uid),
				goqu.C("user_id").Eq(auth.GetRequestUser(r).ID),
			)
			if err != nil {
				api.srv.Status(w, r, http.StatusNotFound)
				return
			}
			ctx = context.WithValue(r.Context(), ctxCollectionKey{}, c)
		}

		// Apply filters
		f := newCollectionForm()
		f.Filters = newContextFilterForm(r.Context())
		f.setCollection(c)
		f.Filters.order = []exp.OrderedExpression{goqu.I("created").Desc()}
		ctx = f.Filters.saveContext(ctx)

		next.ServeHTTP(w, r.Clone(ctx))
	})
}

func (api *apiRouter) withBookmarkList(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := bookmarkList{}

		limit, ok := r.Context().Value(ctxDefaultLimitKey{}).(int)
		if !ok {
			limit = 50
		}

		pf := api.srv.GetPageParams(r, limit)
		if pf == nil {
			api.srv.Status(w, r, http.StatusNotFound)
			return
		}

		ds := Bookmarks.Query().
			Select(
				"b.id", "b.uid", "b.created", "b.updated", "b.state", "b.url", "b.title",
				"b.domain", "b.site", "b.site_name", "b.authors", "b.lang", "b.dir", "b.type",
				"b.is_marked", "b.is_archived",
				"b.labels", "b.description", "b.word_count", "b.duration", "b.file_path", "b.files").
			Where(
				goqu.C("user_id").Table("b").Eq(auth.GetRequestUser(r).ID),
			)

		ds = ds.Order(goqu.I("created").Desc())

		// Filters (search and other filters)
		filters := newContextFilterForm(r.Context())

		// We accept any values coming from a post or get method.
		_ = r.ParseForm()
		forms.UnmarshalValues(filters, r.Form)

		if filters.IsValid() {
			ds = filters.toSelectDataSet(ds)
		}

		ds = ds.
			Limit(uint(pf.Limit())).
			Offset(uint(pf.Offset()))

		if filters.noPagination {
			ds = ds.ClearLimit().ClearOffset()
		}

		var count int64
		var err error
		if count, err = ds.ClearOrder().ClearLimit().ClearOffset().Count(); err != nil {
			if errors.Is(err, ErrBookmarkNotFound) {
				api.srv.TextMessage(w, r, http.StatusNotFound, "not found")
			} else {
				api.srv.Error(w, r, err)
			}
			return
		}

		res.items = []*Bookmark{}
		if err = ds.ScanStructs(&res.items); err != nil {
			api.srv.Error(w, r, err)
			return
		}

		res.Pagination = api.srv.NewPagination(r, int(count), pf.Limit(), pf.Offset())

		ctx := filters.saveContext(r.Context())
		ctx = context.WithValue(ctx, ctxBookmarkListKey{}, res)

		tagers := []server.Etager{res}
		t, ok := r.Context().Value(ctxBookmarkListTagerKey{}).([]server.Etager)
		if ok {
			tagers = append(tagers, t...)
		}

		if r.Method == http.MethodGet {
			api.srv.WriteEtag(w, r, tagers...)
		}
		api.srv.WithCaching(next).ServeHTTP(w, r.Clone(ctx))
	})
}

func (api *apiRouter) withAnnotationList(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := annotationList{}

		limit, ok := r.Context().Value(ctxDefaultLimitKey{}).(int)
		if !ok {
			limit = 50
		}

		pf := api.srv.GetPageParams(r, limit)
		if pf == nil {
			api.srv.Status(w, r, http.StatusNotFound)
			return
		}

		ds := Bookmarks.GetAnnotations().
			Where(
				goqu.C("user_id").Table("b").Eq(auth.GetRequestUser(r).ID),
			)

		ds = ds.
			Limit(uint(pf.Limit())).
			Offset(uint(pf.Offset())).
			Order(goqu.I("annotation_created").Desc())

		var count int64
		var err error

		if count, err = ds.ClearOrder().ClearLimit().ClearOffset().Count(); err != nil {
			api.srv.Error(w, r, err)
			return
		}

		res.Pagination = api.srv.NewPagination(r, int(count), pf.Limit(), pf.Offset())

		res.items = []*annotationQueryResult{}
		if err = ds.ScanStructs(&res.items); err != nil {
			api.srv.Error(w, r, err)
			return
		}
		res.Items = make([]annotationItem, len(res.items))
		for i, item := range res.items {
			res.Items[i] = newAnnotationItem(api.srv, r, item)
		}

		ctx := context.WithValue(r.Context(), ctxAnnotationListKey{}, res)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (api *apiRouter) withLabelList(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ds := Bookmarks.GetLabels().
			Where(
				goqu.C("user_id").Table("b").Eq(auth.GetRequestUser(r).ID),
			)

		f := newLabelSearchForm()
		forms.UnmarshalValues(f, r.URL.Query())
		if f.Get("q").String() != "" {
			q := strings.ReplaceAll(f.Get("q").String(), "*", "%")
			ds = ds.Where(goqu.I("name").Like(q))
		}

		res := []*labelItem{}
		if err := ds.ScanStructs(&res); err != nil {
			api.srv.Error(w, r, err)
			return
		}

		ctx := context.WithValue(r.Context(), ctxLabelListKey{}, res)
		next.ServeHTTP(w, r.Clone(ctx))
	})
}

func (api *apiRouter) withSharedLink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Disable HTTP caching
		api.srv.WriteLastModified(w, r)
		api.srv.WriteEtag(w, r)

		b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
		if b.State != StateLoaded {
			api.srv.Error(w, r, errors.New("bookmark not loaded yet"))
			return
		}

		expires := time.Now().Round(time.Minute).Add(sharingDuration)

		rr, err := encryptID(uint64(b.ID), expires)
		if err != nil {
			api.srv.Error(w, r, err)
			return
		}

		info := sharedBookmarkItem{
			URL:     api.srv.AbsoluteURL(r, "/@b", rr).String(),
			Expires: expires,
			Title:   b.Title,
			ID:      b.UID,
		}
		ctx := context.WithValue(r.Context(), ctxSharedInfoKey{}, info)
		w.Header().Set("Location", info.URL)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bookmarkList is a paginated list of BookmarkItem instances.
type bookmarkList struct {
	items      []*Bookmark
	Pagination server.Pagination
	Items      []bookmarkItem
}

func (bl bookmarkList) GetSumStrings() []string {
	r := []string{}
	for i := range bl.items {
		r = append(r, bl.items[i].Updated.String(), bl.items[i].UID)
	}

	return r
}

// bookmarkItem is a serialized bookmark instance that can
// be used directly on the API or by an HTML template.
type bookmarkItem struct {
	*Bookmark `json:"-"`

	ID            string                   `json:"id"`
	Href          string                   `json:"href"`
	Created       time.Time                `json:"created"`
	Updated       time.Time                `json:"updated"`
	State         BookmarkState            `json:"state"`
	Loaded        bool                     `json:"loaded"`
	URL           string                   `json:"url"`
	Title         string                   `json:"title"`
	SiteName      string                   `json:"site_name"`
	Site          string                   `json:"site"`
	Published     *time.Time               `json:"published,omitempty"`
	Authors       []string                 `json:"authors"`
	Lang          string                   `json:"lang"`
	TextDirection string                   `json:"text_direction"`
	DocumentType  string                   `json:"document_type"`
	Type          string                   `json:"type"`
	HasArticle    bool                     `json:"has_article"`
	Description   string                   `json:"description"`
	IsDeleted     bool                     `json:"is_deleted"`
	IsMarked      bool                     `json:"is_marked"`
	IsArchived    bool                     `json:"is_archived"`
	Labels        []string                 `json:"labels"`
	Annotations   BookmarkAnnotations      `json:"-"`
	Resources     map[string]*bookmarkFile `json:"resources"`
	Embed         string                   `json:"embed,omitempty"`
	EmbedHostname string                   `json:"embed_domain,omitempty"`
	Errors        []string                 `json:"errors,omitempty"`
	Links         BookmarkLinks            `json:"links,omitempty"`

	baseURL            *url.URL
	mediaURL           *url.URL
	annotationTag      string
	annotationCallback func(id string, n *html.Node, index int)
}

// bookmarkFile is a file attached to a bookmark. If the file is
// an image, the "Width" and "Height" values will be filled.
type bookmarkFile struct {
	Src    string `json:"src"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// newBookmarkItem builds a BookmarkItem from a Bookmark instance.
func newBookmarkItem(s *server.Server, r *http.Request, b *Bookmark, base string) bookmarkItem {
	res := bookmarkItem{
		Bookmark:      b,
		ID:            b.UID,
		Href:          s.AbsoluteURL(r, base, b.UID).String(),
		Created:       b.Created,
		Updated:       b.Updated,
		State:         b.State,
		Loaded:        b.State != StateLoading,
		URL:           b.URL,
		Title:         b.Title,
		SiteName:      b.SiteName,
		Site:          b.Site,
		Published:     b.Published,
		Authors:       b.Authors,
		Lang:          b.Lang,
		TextDirection: b.TextDirection,
		DocumentType:  b.DocumentType,
		Description:   b.Description,
		IsDeleted:     deleteBookmarkTask.IsRunning(b.ID),
		IsMarked:      b.IsMarked,
		IsArchived:    b.IsArchived,
		Labels:        make([]string, 0),
		Annotations:   b.Annotations,
		Resources:     make(map[string]*bookmarkFile),
		Links:         b.Links,

		baseURL:       s.AbsoluteURL(r, "/"),
		annotationTag: "rd-annotation",
		annotationCallback: func(id string, n *html.Node, index int) {
			if index == 0 {
				dom.SetAttribute(n, "id", fmt.Sprintf("annotation-%s", id))
			}
			dom.SetAttribute(n, "data-annotation-id-value", id)
		},
	}

	// Set a relative media base URL when we're not querying the API.
	if !strings.HasPrefix(r.URL.EscapedPath(), s.AbsoluteURL(r, "/api/").EscapedPath()) {
		res.baseURL.Scheme = ""
		res.baseURL.Host = ""
	}

	res.mediaURL = res.baseURL.JoinPath("/bm", b.FilePath)

	if b.Labels != nil {
		res.Labels = b.Labels
	}

	switch res.DocumentType {
	case "video":
		res.Type = "video"
	case "image", "photo":
		res.Type = "photo"
	default:
		res.Type = "article"
	}

	for k, v := range b.Files {
		if path.Dir(v.Name) != "img" {
			continue
		}

		f := &bookmarkFile{
			Src: res.mediaURL.String() + "/" + v.Name,
		}

		if v.Size != [2]int{0, 0} {
			f.Width = v.Size[0]
			f.Height = v.Size[1]
		}
		res.Resources[k] = f
	}

	if v, ok := b.Files["props"]; ok {
		res.Resources["props"] = &bookmarkFile{Src: s.AbsoluteURL(r, base, b.UID, "x", v.Name).String()}
	}
	if v, ok := b.Files["log"]; ok {
		res.Resources["log"] = &bookmarkFile{Src: s.AbsoluteURL(r, base, b.UID, "x", v.Name).String()}
	}
	if _, ok := b.Files["article"]; ok {
		res.HasArticle = true
		res.Resources["article"] = &bookmarkFile{Src: s.AbsoluteURL(r, base, b.UID, "article").String()}
	}

	return res
}

// getArticle returns a strings.Reader containing the
// HTML content of a bookmark. Only the body is retrieved.
//
// Note: this method will always return a non nil strings.Reader. In case of error
// it might be empty or the original one if some transformation failed.
// This lets us test for error and log them when needed.
func (bi bookmarkItem) getArticle() (*strings.Reader, error) {
	var err error
	var c *bookmarkContainer
	if c, err = bi.OpenContainer(); err != nil {
		return strings.NewReader(""), err
	}
	defer c.Close()

	if err = c.LoadArticle(); err != nil {
		if os.IsNotExist(err) {
			return strings.NewReader(""), nil
		}
		return strings.NewReader(""), err
	}

	if err = c.ReplaceLinks(
		"./_resources",
		fmt.Sprintf("%s/_resources", bi.mediaURL.String()),
	); err != nil {
		return strings.NewReader(""), err
	}

	if err = c.ExtractBody(); err != nil {
		return strings.NewReader(""), err
	}

	reader := strings.NewReader(c.GetArticle())

	// Add bookmark annotations
	if len(bi.Annotations) > 0 {
		return bi.addAnnotations(reader)
	}

	return reader, nil
}

// setEmbed sets the Embed and EmbedHostname item properties.
// The original embed value must be an iframe. We extract the "src"
// URL and store its hostname that we can later use in the CSP policy.
// A special case for youtube for which we force
// the use of youtube-nocookie.com.
func (bi *bookmarkItem) setEmbed() error {
	if bi.Bookmark.Embed == "" || bi.EmbedHostname != "" {
		return nil
	}
	node, err := html.Parse(strings.NewReader(bi.Bookmark.Embed))
	if err != nil {
		return err
	}
	embed := dom.QuerySelector(node, "iframe,hls,video")
	if embed == nil {
		return nil
	}

	src, err := url.Parse(dom.GetAttribute(embed, "src"))
	if err != nil {
		return err
	}

	// Force youtube iframes to use the "nocookie" variant.
	if src.Host == "www.youtube.com" {
		src.Host = "www.youtube-nocookie.com"
	}

	switch dom.TagName(embed) {
	case "iframe":
		// Set the embed block and its hostname
		dom.SetAttribute(embed, "src", src.String())
		bi.Embed = dom.OuterHTML(embed)
		bi.EmbedHostname = src.Hostname()
	case "hls":
		playerURL := bi.baseURL.JoinPath("/videoplayer")
		playerURL.RawQuery = url.Values{
			"type": {"hls"},
			"src":  {src.String()},
			"w":    {strconv.Itoa(bi.Resources["image"].Width)},
			"h":    {strconv.Itoa(bi.Resources["image"].Height)},
		}.Encode()
		bi.Embed = fmt.Sprintf(
			`<iframe src="%s" width="%d" height="%d" frameborder="0" scrolling="no" sandbox="allow-scripts"></iframe>`,
			playerURL,
			bi.Resources["image"].Width,
			bi.Resources["image"].Height,
		)
	case "video":
		playerURL := bi.baseURL.JoinPath("/videoplayer")
		playerURL.RawQuery = url.Values{
			"src": {src.String()},
			"w":   {strconv.Itoa(bi.Resources["image"].Width)},
			"h":   {strconv.Itoa(bi.Resources["image"].Height)},
		}.Encode()
		bi.Embed = fmt.Sprintf(
			`<iframe src="%s" width="%d" height="%d" frameborder="0" scrolling="no" sandbox="allow-scripts"></iframe>`,
			playerURL,
			bi.Resources["image"].Width,
			bi.Resources["image"].Height,
		)
	}

	return nil
}

// addAnnotations adds the given annotations to the document's content.
// annotations is a parameter for we can use this method to add existing annotations or
// add a new one (and use this method as a validator).
func (bi bookmarkItem) addAnnotations(input *strings.Reader) (*strings.Reader, error) {
	var err error
	var doc *html.Node

	if doc, err = html.Parse(input); err != nil {
		input.Seek(0, 0) //nolint:errcheck
		return input, err
	}
	root := dom.QuerySelector(doc, "body")

	err = bi.Annotations.addToNode(root, bi.annotationTag, bi.annotationCallback)
	if err != nil {
		input.Seek(0, 0) //nolint:errcheck
		return input, err
	}

	buf := new(strings.Builder)
	if err = html.Render(buf, doc); err != nil {
		input.Seek(0, 0) //nolint:errcheck
		return input, err
	}
	reader := strings.NewReader(buf.String())
	return reader, nil
}

type labelItem struct {
	Name  labelString `json:"name"`
	Count int         `json:"count"`
	Href  string      `json:"href"`
}

type labelString string

func (s labelString) Path() string {
	return url.QueryEscape(string(s))
}

type annotationList struct {
	items      []*annotationQueryResult
	Pagination server.Pagination
	Items      []annotationItem
}

type annotationItem struct {
	ID               string    `json:"id"`
	Href             string    `json:"href"`
	Text             string    `json:"text"`
	Created          time.Time `json:"created"`
	BookmarkID       string    `json:"bookmark_id"`
	BookmarkHref     string    `json:"bookmark_href"`
	BookmarkURL      string    `json:"bookmark_url"`
	BookmarkTitle    string    `json:"bookmark_title"`
	BookmarkSiteName string    `json:"bookmark_site_name"`
}

func newAnnotationItem(s *server.Server, r *http.Request, a *annotationQueryResult) annotationItem {
	res := annotationItem{
		ID:               a.ID,
		Href:             s.AbsoluteURL(r, "/api/bookmarks", a.Bookmark.UID, "annotations", a.ID).String(),
		Text:             a.Text,
		Created:          time.Time(a.Created),
		BookmarkID:       a.Bookmark.UID,
		BookmarkHref:     s.AbsoluteURL(r, "/api/bookmarks", a.Bookmark.UID).String(),
		BookmarkURL:      a.Bookmark.URL,
		BookmarkTitle:    a.Bookmark.Title,
		BookmarkSiteName: a.Bookmark.SiteName,
	}
	return res
}

type sharedBookmarkItem struct {
	URL     string    `json:"url"`
	Expires time.Time `json:"expires"`
	Title   string    `json:"title"`
	ID      string    `json:"id"`
}
