// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lithammer/shortuuid/v4"

	"codeberg.org/readeck/readeck/internal/auth"
	"codeberg.org/readeck/readeck/internal/bookmarks/importer"
	"codeberg.org/readeck/readeck/internal/server"
	"codeberg.org/readeck/readeck/pkg/forms"
)

func (h *viewsRouter) bookmarksImportMain(w http.ResponseWriter, r *http.Request) {
	uid := chi.URLParam(r, "uid")

	ctx := server.TC{}
	if uid != "" {
		ctx["Operation"] = importer.ImportBookmarksTask.IsRunning(uid)
	}

	h.srv.RenderTemplate(w, r, 200, "/bookmarks/import/index", ctx)
}

func (h *viewsRouter) bookmarksImport(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")
	if source == "" {
		h.srv.Status(w, r, http.StatusNotFound)
		return
	}

	adapter := importer.LoadAdapter(source)
	if adapter == nil {
		h.srv.Status(w, r, http.StatusNotFound)
		return
	}

	f := adapter.Form()
	if f, ok := f.(forms.Localized); ok {
		f.SetLocale(h.srv.Locale(r))
	}

	templateName := fmt.Sprintf("/bookmarks/import/form-%s", source)
	ctx := server.TC{}
	ctx["Form"] = f

	if r.Method == http.MethodPost {
		forms.Bind(f, r)

		var data []byte
		var err error
		if f.IsValid() {
			data, err = adapter.Params(f)
		}
		if err != nil {
			h.srv.Error(w, r, err)
			return
		}

		if !f.IsValid() {
			h.srv.RenderTemplate(w, r, http.StatusUnprocessableEntity, templateName, ctx)
			return
		}

		// Create the import task
		taskID := shortuuid.New()
		err = importer.ImportBookmarksTask.Run(taskID, importer.ImportParams{
			Source:    source,
			Data:      data,
			UserID:    auth.GetRequestUser(r).ID,
			RequestID: h.srv.GetReqID(r),
		})
		if err != nil {
			h.srv.Error(w, r, err)
			return
		}

		h.srv.Redirect(w, r, "/bookmarks/import", taskID)
		return
	}

	h.srv.RenderTemplate(w, r, 200, templateName, ctx)
}
