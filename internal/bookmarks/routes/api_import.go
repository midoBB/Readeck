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
	"codeberg.org/readeck/readeck/pkg/forms"
)

const maxUploadSize = 4 << 20

func (api *apiRouter) bookmarksImport(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")

	if r.ContentLength > maxUploadSize {
		api.srv.TextMessage(w, r, http.StatusRequestEntityTooLarge, "Payload Too Large")
		return
	}

	adapter := importer.LoadAdapter(source)
	if adapter == nil {
		api.srv.TextMessage(w, r, http.StatusNotFound, fmt.Sprintf(`Import from "%s" does not exist.`, source))
		return
	}

	f := adapter.Form()
	forms.Bind(f, r)

	// If the form is valid, we can load the adapter parameters.
	var data []byte
	var err error
	if f.IsValid() {
		data, err = adapter.Params(f)
	}

	// At this point, any error is fatal.
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	// The adapter's form might have changed so we can only check the form's validity now.
	if !f.IsValid() {
		api.srv.Render(w, r, http.StatusUnprocessableEntity, f)
		return
	}

	// Create the import task
	taskID := shortuuid.New()
	err = importer.ImportBookmarksTask.Run(taskID, importer.ImportParams{
		Source:    source,
		Data:      data,
		UserID:    auth.GetRequestUser(r).ID,
		RequestID: api.srv.GetReqID(r),
	})
	if err != nil {
		api.srv.Error(w, r, err)
		return
	}

	api.srv.TextMessage(w, r, http.StatusAccepted, "import started")
}
