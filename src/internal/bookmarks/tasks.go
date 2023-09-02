// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/bus"
	"codeberg.org/readeck/readeck/internal/db"
	"codeberg.org/readeck/readeck/pkg/archiver"
	"codeberg.org/readeck/readeck/pkg/extract"
	"codeberg.org/readeck/readeck/pkg/extract/contents"
	"codeberg.org/readeck/readeck/pkg/extract/fftr"
	"codeberg.org/readeck/readeck/pkg/extract/meta"
	"codeberg.org/readeck/readeck/pkg/extract/rules"
	"codeberg.org/readeck/readeck/pkg/superbus"
)

var (
	extractPageTask      superbus.Task
	deleteBookmarkTask   superbus.Task
	deleteCollectionTask superbus.Task
	deleteLabelTask      superbus.Task
)

type (
	extractParams struct {
		BookmarkID int
		RequestID  string
		Resources  []multipartResource
	}

	labelDeleteParams struct {
		UserID int
		Name   string
	}
)

func init() {
	bus.OnReady(func() {
		extractPageTask = bus.Tasks().NewTask(
			"bookmark.create",
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res extractParams
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(extractPageHandler),
		)

		deleteBookmarkTask = bus.Tasks().NewTask(
			"bookmark.delete",
			superbus.WithTaskDelay(20),
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res int
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(deleteBookmarkHandler),
		)

		deleteCollectionTask = bus.Tasks().NewTask(
			"collection.delete",
			superbus.WithTaskDelay(20),
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res int
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(deleteCollectionHandler),
		)

		deleteLabelTask = bus.Tasks().NewTask(
			"label.delete",
			superbus.WithTaskDelay(20),
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res labelDeleteParams
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(deleteLabelHandler),
		)
	})
}

func deleteBookmarkHandler(data interface{}) {
	id := data.(int)
	logger := log.WithField("id", id)

	logger.Debug("deleting bookmark")
	b, err := Bookmarks.GetOne(goqu.C("id").Eq(id))
	if err != nil {
		logger.WithError(err).Error("bookmark retrieve")
		return
	}

	if err := b.Delete(); err != nil {
		logger.WithError(err).Error("bookmark removal")
		return
	}

	logger.Info("bookmark removed")
}

func deleteCollectionHandler(data interface{}) {
	id := data.(int)
	logger := log.WithField("id", id)

	logger.Debug("deleting collection")

	c, err := Collections.GetOne(goqu.C("id").Eq(id))
	if err != nil {
		logger.WithError(err).Error("collection retrieve")
		return
	}

	if err := c.Delete(); err != nil {
		logger.WithError(err).Error("collection removal")
		return
	}

	logger.Info("collection removed")
}

func deleteLabelHandler(data interface{}) {
	params := data.(labelDeleteParams)
	logger := log.WithFields(log.Fields{
		"user":  params.UserID,
		"label": params.Name,
	})
	logger.Debug("deleting label")

	u, err := users.Users.GetOne(goqu.C("id").Eq(params.UserID))
	if err != nil {
		logger.WithError(err).Error("user retrieve")
		return
	}

	if _, err = Bookmarks.RenameLabel(u, params.Name, ""); err != nil {
		logger.WithError(err).Error("label remove")
		return
	}

	logger.Info("label removed")
}

func extractPageHandler(data interface{}) {
	var b *Bookmark
	var err error

	params := data.(extractParams)

	saved := false
	logger := log.WithFields(log.Fields{
		"@id":         params.RequestID,
		"bookmark_id": params.BookmarkID,
	})
	logger.Debug("starting extraction")

	defer func() {
		if b == nil {
			return
		}

		// Recover from any error that could have arose
		if r := recover(); r != nil {
			logger.WithField("recover", r).Error("error during extraction")
			debug.PrintStack()
			b.State = StateError
			b.Errors = append(b.Errors, fmt.Sprintf("%v", r))
			saved = false
		}

		// Never stay hanging
		if b.State == StateLoading {
			b.State = StateLoaded
			saved = false
		}

		// Then save the whole thing
		if !saved {
			b.Save()
		}
		runtime.GC()
	}()

	b, err = Bookmarks.GetOne(goqu.C("id").Eq(params.BookmarkID))
	if err != nil {
		logger.WithError(err).Error()
		return
	}

	proxyList := make([]extract.ProxyMatcher, len(configs.Config.Extractor.ProxyMatch))
	for i, x := range configs.Config.Extractor.ProxyMatch {
		proxyList[i] = x
	}

	ex, err := extract.New(
		b.URL,
		extract.SetLogFields(&log.Fields{
			"@id":         params.RequestID,
			"bookmark_id": b.ID,
		}),
		extract.SetDeniedIPs(configs.ExtractorDeniedIPs()),
		extract.SetProxyList(proxyList),
	)
	if err != nil {
		logger.WithError(err).Error()
		return
	}

	for _, x := range params.Resources {
		// Inject resource in client's cache
		ex.AddToCache(x.URL, x.Headers, bytes.NewReader(x.Data))
	}

	ex.AddProcessors(
		meta.ExtractMeta,
		meta.ExtractOembed,
		rules.ApplyRules,
		meta.SetDropProperties,
		meta.ExtractFavicon,
		meta.ExtractPicture,
		fftr.LoadConfiguration,
		fftr.ReplaceStrings,
	)

	// Check if main page is in cached resources
	if !ex.IsInCache(b.URL) {
		ex.AddProcessors(fftr.FindContentPage, fftr.FindNextPage)
	}

	ex.AddProcessors(
		fftr.ExtractAuthor,
		fftr.ExtractDate,
		fftr.ExtractBody,
		fftr.StripTags,
		fftr.GoToNextPage,
		contents.Readability(),
		CleanDomProcessor,
		extractLinksProcessor,
		contents.Text,
		saveBookmark(b, &saved),
		fetchLinksProcessor(b),
	)

	ex.Run()
}

// saveBookmark is one last step of the extraction process, it saves the bookmark
// and marks it ready for reading.
// Other steps can still perform tasks later.
func saveBookmark(b *Bookmark, saved *bool) extract.Processor {
	return func(m *extract.ProcessMessage, next extract.Processor) extract.Processor {
		if m.Step() != extract.StepDone {
			return next
		}

		ex := m.Extractor
		var err error
		drop := ex.Drop()
		if drop == nil {
			return next
		}

		b.Updated = time.Now()
		b.URL = drop.UnescapedURL()
		b.State = StateLoaded
		b.Domain = drop.Domain
		b.Site = drop.URL.Hostname()
		b.SiteName = drop.Site
		b.Authors = db.Strings{}
		b.Lang = drop.Lang
		b.DocumentType = drop.DocumentType
		b.Description = drop.Description
		b.Text = ex.Text
		b.WordCount = len(strings.Fields(b.Text))

		if b.Title == "" {
			b.Title = drop.Title
		}

		for _, x := range drop.Authors {
			b.Authors = append(b.Authors, x)
		}

		for _, x := range ex.Errors() {
			b.Errors = append(b.Errors, x.Error())
		}

		if !drop.Date.IsZero() {
			b.Published = &drop.Date
		}

		if drop.IsMedia() {
			b.Embed = drop.Meta.LookupGet("oembed.html")
		}

		b.Links = GetExtractedLinks(ex.Context)

		// Run the archiver
		var arc *archiver.Archiver
		logEntry := log.NewEntry(ex.GetLogger()).WithFields(*ex.LogFields)
		if len(ex.HTML) > 0 && ex.Drop().IsHTML() {
			arc, err = newArchive(context.TODO(), ex)
			if err != nil {
				logEntry.WithError(err).Error("archiver error")
			}
		}

		// Create the zip file
		err = createZipFile(b, ex, arc)
		if err != nil {
			// If something goes really wrong, cleanup after ourselves
			b.Errors = append(b.Errors, err.Error())
			b.removeFiles()
			b.FilePath = ""
			b.Files = BookmarkFiles{}
		}

		// All good? Save now
		if err := b.Save(); err != nil {
			log.WithError(err).Error()
			return next
		}
		*saved = true
		return next
	}
}

// fetchLinksProcessor retrieves the link list (from extractLinksProcessor) and
// process all of them to get some information (content type, title when possible...)
// The link list is then saved into the bookmark.
// This processor MUST run after saveBookmark.
func fetchLinksProcessor(b *Bookmark) extract.Processor {
	return func(m *extract.ProcessMessage, next extract.Processor) extract.Processor {
		if m.Step() != extract.StepDone {
			return next
		}

		links, ok := m.Extractor.Context.Value(ctxExtractLinksKey).(BookmarkLinks)
		if !ok {
			return next
		}

		g, _ := errgroup.WithContext(context.TODO())
		g.SetLimit(10)
		for i := range links {
			i := i
			g.Go(func() error {
				log.WithField("url", links[i].URL).Debug("extract link")
				URL, err := url.Parse(links[i].URL)
				if err != nil {
					return err
				}
				// d := seen[links[i].URL]
				d := extract.NewDrop(URL)
				err = d.Load(m.Extractor.Client())

				if err != nil {
					log.WithField("url", d.URL).WithError(err).Warn("extract link error")
				}

				links[i].ContentType = d.ContentType
				links[i].IsPage = d.IsHTML()

				if !links[i].IsPage {
					return nil
				}

				node, err := html.Parse(bytes.NewReader(d.Body))
				if err != nil {
					log.WithField("url", d.URL).WithError(err).Warn("extract link error")
					return nil
				}
				meta := meta.ParseMeta(node)
				title := meta.LookupGet("graph.title", "tiwtter.title", "html.title")
				log.WithField("url", d.URL.String()).WithField("title", title).Debug("link")

				links[i].Title = title
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			log.WithError(err).Error("extract links")
		}

		links = slices.CompactFunc(links, func(a, b BookmarkLink) bool {
			return a.URL == b.URL
		})

		if len(links) == 0 {
			return next
		}

		if err := b.Update(map[string]any{"links": links}); err != nil {
			log.WithError(err).Error()
		}

		return next
	}
}

func createZipFile(b *Bookmark, ex *extract.Extractor, arc *archiver.Archiver) error {
	// Fail fast
	fileURL, err := b.getBaseFileURL()
	if err != nil {
		return err
	}
	zipFile := filepath.Join(StoragePath(), fileURL+".zip")

	b.FilePath = fileURL
	b.Files = BookmarkFiles{}

	// Create the zip file
	z, err := newZipper(zipFile)
	if err != nil {
		return err
	}
	defer func() {
		err := z.close()
		if err != nil {
			panic(err)
		}
	}()

	// Add images to the zipfile
	if err = z.addDirectory("img"); err != nil {
		return err
	}

	for k, p := range ex.Drop().Pictures {
		name := path.Join("img", p.Name(k))
		if err = z.addFile(name, p.Bytes()); err != nil {
			return err
		}
		b.Files[k] = &BookmarkFile{name, p.Type, p.Size}
	}

	// Add HTML content
	if arc != nil && len(arc.Result) > 0 {
		if err = z.addCompressedFile("index.html", arc.Result); err != nil {
			return err
		}
		b.Files["article"] = &BookmarkFile{Name: "index.html"}
	}

	// Add assets
	if arc != nil && len(arc.Cache) > 0 {
		if err = z.addDirectory(resourceDirName); err != nil {
			return err
		}

		for uri, asset := range arc.Cache {
			fname := path.Join(resourceDirName, getURLfilename(uri, asset.ContentType))
			if err = z.addFile(fname, asset.Data); err != nil {
				return err
			}
		}
	}

	// Add the log
	if err = z.addCompressedFile("log", []byte(strings.Join(ex.Logs, "\n"))); err != nil {
		return err
	}
	b.Files["log"] = &BookmarkFile{Name: "log"}

	// Add the metadata
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	if err = enc.Encode(ex.Drop()); err != nil {
		return err
	}
	if err = z.addCompressedFile("props.json", buf.Bytes()); err != nil {
		return err
	}
	b.Files["props"] = &BookmarkFile{Name: "props.json"}

	return nil
}
