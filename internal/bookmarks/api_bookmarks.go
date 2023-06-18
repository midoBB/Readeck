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

	"github.com/readeck/readeck/internal/auth"
	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/annotate"
	"github.com/readeck/readeck/pkg/forms"
	"github.com/readeck/readeck/pkg/zipfs"
)

type (
	ctxBookmarkKey          struct{}
	ctxBookmarkListKey      struct{}
	ctxBookmarkListTagerKey struct{}
	ctxLabelKey             struct{}
	ctxLabelListKey         struct{}
	ctxFiltersKey           struct{}
	ctxDefaultLimitKey      struct{}
)

// bookmarkList renders a paginated list of the connected
// user bookmarks in JSON.
func (api *apiRouter) bookmarkList(w http.ResponseWriter, r *http.Request) {
	bl := r.Context().Value(ctxBookmarkListKey{}).(bookmarkList)

	bl.Items = make([]bookmarkItem, len(bl.items))
	for i, item := range bl.items {
		bl.Items[i] = newBookmarkItem(api.srv, r, item, ".")
	}

	api.srv.SendPaginationHeaders(w, r, bl.Pagination.TotalCount, bl.Pagination.Limit, bl.Pagination.Offset)
	api.srv.Render(w, r, http.StatusOK, bl.Items)
}

// bookmarkInfo renders a given bookmark items in JSON.
func (api *apiRouter) bookmarkInfo(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	item := newBookmarkItem(api.srv, r, b, "./..")

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
	ds = Bookmarks.AddLabelFilter(ds, []string{label})
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
	f := newLabelForm(auth.GetRequestUser(r).ID)
	forms.Bind(f, r)

	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusBadRequest, f)
		return
	}

	ids, err := f.rename(label)
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}
	if len(ids) == 0 {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}
}

func (api *apiRouter) annotationList(w http.ResponseWriter, r *http.Request) {
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
			api.srv.WriteLastModified(w, b)
			api.srv.WriteEtag(w, b)
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
				"b.domain", "b.site", "b.site_name", "b.authors", "b.lang", "b.type",
				"b.is_marked", "b.is_archived",
				"b.labels", "b.description", "b.word_count", "b.file_path", "b.files").
			Where(
				goqu.C("user_id").Table("b").Eq(auth.GetRequestUser(r).ID),
			)

		ds = ds.Order(goqu.I("created").Desc())

		// Filters (search and other filters)
		filters := newContextFilterForm(r.Context())
		forms.UnmarshalValues(filters, r.URL.Query())
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
			api.srv.WriteEtag(w, tagers...)
		}
		api.srv.WithCaching(next).ServeHTTP(w, r.Clone(ctx))
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

	ID           string                   `json:"id"`
	Href         string                   `json:"href"`
	Created      time.Time                `json:"created"`
	Updated      time.Time                `json:"updated"`
	State        BookmarkState            `json:"state"`
	Loaded       bool                     `json:"loaded"`
	URL          string                   `json:"url"`
	Title        string                   `json:"title"`
	SiteName     string                   `json:"site_name"`
	Site         string                   `json:"site"`
	Published    *time.Time               `json:"published,omitempty"`
	Authors      []string                 `json:"authors"`
	Lang         string                   `json:"lang"`
	DocumentType string                   `json:"document_type"`
	Type         string                   `json:"type"`
	Description  string                   `json:"description"`
	IsDeleted    bool                     `json:"is_deleted"`
	IsMarked     bool                     `json:"is_marked"`
	IsArchived   bool                     `json:"is_archived"`
	Labels       []string                 `json:"labels"`
	Resources    map[string]*bookmarkFile `json:"resources"`
	Embed        string                   `json:"embed,omitempty"`
	Errors       []string                 `json:"errors,omitempty"`

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
		Bookmark:     b,
		ID:           b.UID,
		Href:         s.AbsoluteURL(r, base, b.UID).String(),
		Created:      b.Created,
		Updated:      b.Updated,
		State:        b.State,
		Loaded:       b.State != StateLoading,
		URL:          b.URL,
		Title:        b.Title,
		SiteName:     b.SiteName,
		Site:         b.Site,
		Published:    b.Published,
		Authors:      b.Authors,
		Lang:         b.Lang,
		DocumentType: b.DocumentType,
		Description:  b.Description,
		IsDeleted:    deleteBookmarkTask.IsRunning(b.ID),
		IsMarked:     b.IsMarked,
		IsArchived:   b.IsArchived,
		Labels:       make([]string, 0),
		Resources:    make(map[string]*bookmarkFile),

		mediaURL:      s.AbsoluteURL(r, "/bm", b.FilePath),
		annotationTag: "rd-annotation",
		annotationCallback: func(id string, n *html.Node, index int) {
			if index == 0 {
				dom.SetAttribute(n, "id", fmt.Sprintf("annotation-%s", id))
			}
			dom.SetAttribute(n, "data-annotation-id-value", id)
		},
	}

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
	if c, err = bi.openContainer(); err != nil {
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

// addAnnotations adds the given annotations to the document's content.
// annotations is a parameter for we can use this method to add existing annotations or
// add a new one (and use this method as a validator)
func (bi bookmarkItem) addAnnotations(input *strings.Reader) (*strings.Reader, error) {
	var err error
	var doc *html.Node

	if doc, err = html.Parse(input); err != nil {
		input.Seek(0, 0)
		return input, err
	}
	root := dom.QuerySelector(doc, "body")

	err = bi.Annotations.addToNode(root, bi.annotationTag, bi.annotationCallback)
	if err != nil {
		input.Seek(0, 0)
		return input, err
	}

	buf := new(strings.Builder)
	if err = html.Render(buf, doc); err != nil {
		input.Seek(0, 0)
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
