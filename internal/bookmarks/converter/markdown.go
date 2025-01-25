// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package converter

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
	"github.com/PuerkitoBio/goquery"
	"github.com/gabriel-vasile/mimetype"

	"codeberg.org/readeck/readeck/internal/bookmarks"
	"codeberg.org/readeck/readeck/pkg/accept"
	"codeberg.org/readeck/readeck/pkg/utils"
)

// MarkdownExporter is an content exporter that produces markdown.
type MarkdownExporter struct {
	HTMLConverter
	baseURL      *url.URL
	mediaBaseURL *url.URL
}

var ctxExportTypeKey = contextKey{"export-type"}

// NewMarkdownExporter returns a new [MarkdownExporter] instance.
func NewMarkdownExporter(baseURL *url.URL, mediaBaseURL *url.URL) MarkdownExporter {
	return MarkdownExporter{
		HTMLConverter: HTMLConverter{},
		baseURL:       baseURL,
		mediaBaseURL:  mediaBaseURL,
	}
}

// Export implement [Exporter].
// It can write text only articles (the default) separated by an horizontal rule.
// If the request contains "Accept: multipart/alternative", it returns a multipart response
// that contains images for the exported bookmarks.
func (e MarkdownExporter) Export(ctx context.Context, w io.Writer, r *http.Request, bookmarks []*bookmarks.Bookmark) error {
	converter := html2md.NewConverter("", true, nil)
	converter.Use(plugin.Strikethrough(""))
	converter.Use(plugin.Table())
	converter.Use(plugin.GitHubFlavored())
	converter.Use(mdAnnotation())

	ctx = WithAnnotationTag(ctx, "rd-annotation", nil)

	accepted := accept.NegotiateContentType(r.Header, []string{"text/markdown", "multipart/alternative"}, "text/markdown")
	switch accepted {
	case "multipart/alternative":
		return e.exportMultipart(ctx, w, converter, bookmarks)
	default:
		return e.exportTextOnly(ctx, w, converter, bookmarks)
	}
}

func (e MarkdownExporter) exportTextOnly(ctx context.Context, w io.Writer, converter *html2md.Converter, bookmarks []*bookmarks.Bookmark) error {
	if w, ok := w.(http.ResponseWriter); ok {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	}

	for i, b := range bookmarks {
		c := WithURLReplacer(ctx, "./_resources",
			e.mediaBaseURL.JoinPath(b.FilePath, "_resources").String(),
		)
		if i > 0 {
			fmt.Fprint(w, "\n------------------------------------------------------------\nn") //nolint:errcheck
		}
		if err := e.writeArticle(c, w, converter, b, len(bookmarks) == 1); err != nil {
			slog.Error("export", slog.Any("err", err))
		}
	}
	return nil
}

func (e MarkdownExporter) exportMultipart(ctx context.Context, w io.Writer, converter *html2md.Converter, bookmarks []*bookmarks.Bookmark) error {
	mp := multipart.NewWriter(w)
	if w, ok := w.(http.ResponseWriter); ok {
		w.Header().Set("Content-Type", `multipart/alternative; boundary="`+mp.Boundary()+`"`)
	}
	defer mp.Close() //nolint:errcheck

	ctx = WithURLReplacer(ctx, "./_resources", ".")
	ctx = context.WithValue(ctx, ctxExportTypeKey, "multipart")

	for _, b := range bookmarks {
		if err := func() error {
			slug := utils.Slug(b.Title)
			part, err := mp.CreatePart(textproto.MIMEHeader{
				"BaseName":            []string{slug},
				"Bookmark-Id":         []string{b.UID},
				"Content-Type":        []string{"text/markdown; charset=utf-8"},
				"Content-Disposition": []string{`attachment; filename="` + slug + `.md"`},
				"Date":                []string{b.Created.Format(time.RFC3339)},
			})
			if err != nil {
				return err
			}
			if err := e.writeArticle(ctx, part, converter, b, true); err != nil {
				return err
			}

			bc, err := b.OpenContainer()
			if err != nil {
				return err
			}
			defer bc.Close()

			// Fetch the image
			if img, ok := b.Files["image"]; ok {
				if z, ok := bc.Lookup(img.Name); ok {
					z.Name = e.getImageURL(ctx, b, z.Name)
					if err := e.writeResource(mp, z, b); err != nil {
						return err
					}
				}
			}

			// Fetch all resources
			for _, x := range bc.ListResources() {
				if err := e.writeResource(mp, x, b); err != nil {
					return err
				}
			}

			return nil
		}(); err != nil {
			slog.Error("export", slog.Any("err", err))
			continue
		}
	}

	return nil
}

func (e MarkdownExporter) getImageURL(ctx context.Context, b *bookmarks.Bookmark, name string) string {
	if s, _ := ctx.Value(ctxExportTypeKey).(string); s == "multipart" {
		return b.UID + "-" + path.Base(name)
	}
	return e.mediaBaseURL.JoinPath(b.FilePath, "img", path.Base(name)).String()
}

func (e MarkdownExporter) writeArticle(ctx context.Context, w io.Writer, converter *html2md.Converter, b *bookmarks.Bookmark, withMeta bool) error {
	r, err := e.GetArticle(ctx, b)
	if err != nil {
		return err
	}

	buf, err := converter.ConvertReader(r)
	if err != nil {
		return err
	}

	intro := new(bytes.Buffer)
	if withMeta {
		fmt.Fprintln(intro, "---")
		fmt.Fprintln(intro, "title: "+b.Title)
		fmt.Fprintln(intro, "saved: "+b.Created.Format(time.DateOnly))
		if b.Published != nil {
			fmt.Fprintln(intro, "published: "+b.Published.Format(time.DateOnly))
		}
		fmt.Fprintln(intro, "website: "+b.Site)
		fmt.Fprintln(intro, "source: "+b.URL)
		if len(b.Authors) > 0 {
			authors, _ := json.Marshal(b.Authors)
			fmt.Fprintln(intro, "authors: "+string(authors))
		}
		if len(b.Labels) > 0 {
			labels, _ := json.Marshal(b.Labels)
			fmt.Fprintln(intro, "tag: "+string(labels))
		}

		fmt.Fprintln(intro, "---")
	}

	fmt.Fprintf(intro, "# %s\n\n", b.Title)

	if img, ok := b.Files["image"]; ok {
		fmt.Fprintf(intro, "![](%s)\n\n", e.getImageURL(ctx, b, img.Name))
	}

	if b.DocumentType == "video" {
		fmt.Fprintf(intro, "[Video on %s](%s)\n\n", b.SiteName, b.URL)
	}

	_, err = io.Copy(w, io.MultiReader(intro, &buf))
	return err
}

func (e MarkdownExporter) writeResource(mp *multipart.Writer, resource *zip.File, b *bookmarks.Bookmark) error {
	r, err := resource.Open()
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	buf := new(bytes.Buffer)
	mtype, err := mimetype.DetectReader(io.TeeReader(r, buf))
	if err != nil {
		return err
	}

	part, err := mp.CreatePart(textproto.MIMEHeader{
		"Bookmark-Id":         []string{b.UID},
		"Filename":            []string{path.Base(resource.Name)},
		"Content-Disposition": []string{`attachment; filename="` + path.Base(resource.Name) + `"`},
		"Content-Type":        []string{mtype.String()},
	})
	if err != nil {
		return err
	}

	_, err = io.Copy(part, io.MultiReader(buf, r))

	return err
}

// mdAnnotation is an html-to-markdown plugin that converts rd-annotation tags
// to "=={content}==" form, that's compatible with at least Obsidian.
func mdAnnotation() html2md.Plugin {
	return func(_ *html2md.Converter) []html2md.Rule {
		return []html2md.Rule{
			{
				Filter: []string{"rd-annotation"},
				Replacement: func(content string, selec *goquery.Selection, _ *html2md.Options) *string {
					content = strings.TrimSpace(content)
					if content == "" {
						return &content
					}
					content = "==" + content + "=="
					content = html2md.AddSpaceIfNessesary(selec, content)
					return &content
				},
			},
		}
	}
}
