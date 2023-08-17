package bookmarks

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
)

func (api *apiRouter) exportBookmarksMD(w http.ResponseWriter, r *http.Request, bookmarks ...*Bookmark) {
	if len(bookmarks) == 0 {
		api.srv.Status(w, r, http.StatusNotFound)
		return
	}

	converter := html2md.NewConverter("", true, nil)

	converter.Use(plugin.Strikethrough(""))
	converter.Use(plugin.Table())
	converter.Use(plugin.GitHubFlavored())

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")

	for i, b := range bookmarks {
		if i > 0 {
			w.Write([]byte("\n------------------------------------------------------------\n\n"))
		}

		bi := newBookmarkItem(api.srv, r, b, "")
		w.Write([]byte(fmt.Sprintf("# %s\n\n", bi.Title)))

		var resultBuffer bytes.Buffer

		switch bi.Type {
		case "video":
			fmt.Fprintf(&resultBuffer, "[Video on %s](%s)", bi.SiteName, bi.URL)
		case "photo":
			fmt.Fprintf(&resultBuffer, "![Image on %s](%s)", bi.SiteName, bi.Resources["image"].Src)
		default:
			buf, err := bi.getArticle()
			if err != nil {
				continue
			}

			resultBuffer, err = converter.ConvertReader(buf)
			if err != nil {
				api.srv.Error(w, r, err)
				return
			}
		}
		io.Copy(w, &resultBuffer)
		w.Write([]byte("\n"))

	}
}
