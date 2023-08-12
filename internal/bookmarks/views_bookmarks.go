package bookmarks

import (
	"context"
	"net/http"
	"net/url"
	"os"

	"github.com/readeck/readeck/internal/auth"
	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/forms"
)

type (
	ctxBaseContextKey struct{}
)

func (h *viewsRouter) withBaseContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count, err := Bookmarks.CountAll(auth.GetRequestUser(r))
		if err != nil {
			h.srv.Error(w, r, err)
			return
		}

		c := server.TC{
			"Count": count,
		}

		ctx := context.WithValue(r.Context(), ctxBaseContextKey{}, c)

		next.ServeHTTP(w, r.Clone(ctx))
	})
}

func (h *viewsRouter) bookmarkList(w http.ResponseWriter, r *http.Request) {
	f := newCreateForm(auth.GetRequestUser(r).ID, h.srv.GetReqID(r))

	// POST => create a new bookmark
	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			if b, err := f.createBookmark(); err != nil {
				h.srv.Log(r).Error(err)
			} else {
				redir := []string{"/bookmarks"}
				if h.srv.IsTurboRequest(r) {
					redir = append(redir, "unread")
				} else {
					redir = append(redir, b.UID)
				}
				h.srv.Redirect(w, r, redir...)
				return

			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	// Retrieve the bookmark list
	bl := r.Context().Value(ctxBookmarkListKey{}).(bookmarkList)

	bl.Items = make([]bookmarkItem, len(bl.items))
	for i, item := range bl.items {
		bl.Items[i] = newBookmarkItem(h.srv, r, item, ".")
	}

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Form"] = f
	ctx["Pagination"] = bl.Pagination
	ctx["Bookmarks"] = bl.Items

	if filters, ok := r.Context().Value(ctxFiltersKey{}).(*filterForm); ok {
		ctx["Filters"] = filters
	}

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/index", ctx)
}

func (h *viewsRouter) bookmarkInfo(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	user := auth.GetRequestUser(r)
	item := newBookmarkItem(h.srv, r, b, "")
	item.Embed = b.Embed
	item.Errors = b.Errors

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Item"] = item

	var err error
	ctx["HTML"], err = item.getArticle()
	if err != nil {
		h.srv.Log(r).Error(err)
	}

	// Load bookmark debug information if the user needs them.
	if user.Settings.DebugInfo {
		c, err := b.OpenContainer()
		if err != nil && !os.IsNotExist(err) {
			h.srv.Error(w, r, err)
			return
		}

		if c != nil {
			defer c.Close()

			for k, x := range map[string]string{
				"_props": "props.json",
				"_log":   "log",
			} {
				if r, err := c.GetFile(x); err != nil {
					ctx[k] = err.Error()
				} else {
					ctx[k] = string(r)
				}
			}
		}
	}

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/bookmark", ctx)
}

func (h *viewsRouter) bookmarkUpdate(w http.ResponseWriter, r *http.Request) {
	f := newUpdateForm()
	forms.Bind(f, r)

	if !f.IsValid() {
		h.srv.Render(w, r, http.StatusBadRequest, f)
		return
	}

	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)

	if _, err := f.update(b); err != nil {
		h.srv.Error(w, r, err)
		return
	}

	redir := "/bookmarks/" + b.UID
	if f.Get("_to").String() != "" {
		redir = f.Get("_to").String()
	}

	h.srv.Redirect(w, r, redir)
}

func (h *viewsRouter) bookmarkDelete(w http.ResponseWriter, r *http.Request) {
	b := r.Context().Value(ctxBookmarkKey{}).(*Bookmark)
	f := newDeleteForm()
	forms.Bind(f, r)

	if err := b.Update(map[string]interface{}{}); err != nil {
		h.srv.Error(w, r, err)
		return
	}

	if err := f.trigger(b); err != nil {
		h.srv.Error(w, r, err)
		return
	}

	redir := "/bookmarks"
	if f.Get("_to").String() != "" {
		redir = f.Get("_to").String()
	}

	h.srv.Redirect(w, r, redir)
}

func (h *viewsRouter) labelList(w http.ResponseWriter, r *http.Request) {
	labels := r.Context().Value(ctxLabelListKey{}).([]*labelItem)

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Labels"] = labels

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/labels", ctx)
}

func (h *viewsRouter) labelInfo(w http.ResponseWriter, r *http.Request) {
	bl := r.Context().Value(ctxBookmarkListKey{}).(bookmarkList)
	label := r.Context().Value(ctxLabelKey{}).(string)

	if bl.Pagination.TotalCount == 0 {
		h.srv.Status(w, r, http.StatusNotFound)
		return
	}

	// POST, update label name
	if r.Method == http.MethodPost {
		f := newLabelForm()
		forms.Bind(f, r)

		if f.IsValid() {
			_, err := Bookmarks.RenameLabel(auth.GetRequestUser(r), label, f.Get("name").String())
			if err != nil {
				h.srv.Error(w, r, err)
				return
			}

			// We can't use redirect here, since we must escape the label
			redir := h.srv.AbsoluteURL(r, "/bookmarks/labels/")
			redir.Path += url.QueryEscape(f.Get("name").String())
			w.Header().Set("Location", redir.String())
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	bl.Items = make([]bookmarkItem, len(bl.items))
	for i, item := range bl.items {
		bl.Items[i] = newBookmarkItem(h.srv, r, item, ".")
	}

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Label"] = label
	ctx["Pagination"] = bl.Pagination
	ctx["Bookmarks"] = bl.Items

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/label", ctx)
}
