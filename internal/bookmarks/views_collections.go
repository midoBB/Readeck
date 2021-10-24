package bookmarks

import (
	"net/http"

	"github.com/readeck/readeck/internal/auth"
	"github.com/readeck/readeck/internal/server"
	"github.com/readeck/readeck/pkg/forms"
)

func (h *viewsRouter) collectionList(w http.ResponseWriter, r *http.Request) {
	cl := r.Context().Value(ctxCollectionListKey{}).(collectionList)
	cl.Items = make([]collectionItem, len(cl.items))
	for i, item := range cl.items {
		cl.Items[i] = newCollectionItem(h.srv, r, item, ".")
	}

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Collections"] = cl.Items

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/collection_list", ctx)
}

func (h *viewsRouter) collectionCreate(w http.ResponseWriter, r *http.Request) {
	f := newCollectionForm()

	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			c, err := f.createCollection(auth.GetRequestUser(r).ID)
			if err != nil {
				h.srv.Log(r).Error(err)
			} else {
				h.srv.Redirect(w, r, "./..", c.UID)
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Form"] = f

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/collection_create", ctx)
}

func (h *viewsRouter) collectionInfo(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ctxCollectionKey{}).(*Collection)
	item := newCollectionItem(h.srv, r, c, "./..")

	f := newCollectionForm()
	f.setCollection(c)

	if r.Method == http.MethodPost {
		forms.Bind(f, r)
		if f.IsValid() {
			if _, err := f.updateCollection(c); err != nil {
				h.srv.Log(r).Error(err)
			} else {
				h.srv.AddFlash(w, r, "success", "Collection updated")
				h.srv.Redirect(w, r, c.UID+"?edit=1")
				return
			}
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	bl := r.Context().Value(ctxBookmarkListKey{}).(bookmarkList)
	bl.Items = make([]bookmarkItem, len(bl.items))
	for i, item := range bl.items {
		bl.Items[i] = newBookmarkItem(h.srv, r, item, ".")
	}

	ctx := r.Context().Value(ctxBaseContextKey{}).(server.TC)
	ctx["Editing"] = r.URL.Query().Get("edit") == "1"
	ctx["Item"] = item
	ctx["Form"] = f
	ctx["Pagination"] = bl.Pagination
	ctx["Bookmarks"] = bl.Items

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/collection", ctx)
}

func (h *viewsRouter) collectionDelete(w http.ResponseWriter, r *http.Request) {
	f := newCollectionDeleteForm()
	f.Get("_to").Set("/bookmarks/collections")
	forms.Bind(f, r)

	c := r.Context().Value(ctxCollectionKey{}).(*Collection)

	// This update forces cache invalidation
	if err := c.Update(map[string]interface{}{}); err != nil {
		h.srv.Error(w, r, err)
		return
	}
	f.trigger(c)
	h.srv.Redirect(w, r, f.Get("_to").String())
}
