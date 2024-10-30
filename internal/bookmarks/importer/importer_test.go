// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"codeberg.org/readeck/readeck/internal/bookmarks/importer"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/forms"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

type adapterTest struct {
	adapter importer.ImportLoader
	data    string
	assert  func(test *adapterTest, require *require.Assertions, f forms.Binder, data []byte)
}

func TestFileAdapters(t *testing.T) {
	tests := []adapterTest{
		{
			importer.LoadAdapter("text"),
			"foo\n",
			func(_ *adapterTest, require *require.Assertions, f forms.Binder, _ []byte) {
				require.False(f.IsValid())
				require.Equal("Empty or invalid import file", f.Get("data").Errors.Error())
			},
		},
		{
			importer.LoadAdapter("text"),
			`
			https://example.org/#test
			https://example.net/
			test
			####
			ftp://example.net/
			https://example.net/#foo
			`,
			func(test *adapterTest, require *require.Assertions, f forms.Binder, data []byte) {
				require.True(f.IsValid())
				adapter := test.adapter.(importer.ImportWorker)
				err := adapter.LoadData(data)
				require.NoError(err)

				items := []string{}
				for {
					item, err := adapter.Next()
					if err == io.EOF {
						break
					}
					require.NoError(err)
					items = append(items, item.URL())
				}
				require.Equal([]string{
					"https://example.net/",
					"https://example.org/",
				}, items)
			},
		},
		{
			importer.LoadAdapter("browser"),
			``,
			func(_ *adapterTest, require *require.Assertions, f forms.Binder, _ []byte) {
				require.False(f.IsValid())
				require.Equal("Empty or invalid import file", f.Get("data").Errors.Error())
			},
		},
		{
			importer.LoadAdapter("browser"),
			`
			<!DOCTYPE NETSCAPE-Bookmark-file-1>
			<!-- This is an automatically generated file.
				It will be read and overwritten.
				DO NOT EDIT! -->
			<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
			<TITLE>Bookmarks</TITLE>
			<H1>Bookmarks</H1>
			<DL><p>
				<DT><H3 ADD_DATE="1624868914" LAST_MODIFIED="0" PERSONAL_TOOLBAR_FOLDER="true">Bookmarks bar</H3>
				<DL><p>
					<DT><A HREF="https://www.mozilla.org/en-US/firefox/central/" ADD_DATE="1576652979" ICON="data:image/png;base64,iVBORw0KGgoAAAANSUh">Getting Started</A>
					<DT><A HREF="http://blog.mozilla.com/" ADD_DATE="1601411565" TAGS="mozilla,blog , test " TOREAD="0">Mozilla News</A>
				</DL><p>
				<DT><H3 ADD_DATE="1713598064" LAST_MODIFIED="0">Imported</H3>
				<DL><p>
					<DT><H3 ADD_DATE="1713598064" LAST_MODIFIED="0">Misc</H3>
					<DL><p>
						<DT><A HREF="https://example.net/#test" ADD_DATE="1385462299">Example.net</A>
						<DT><A HREF="https://example.org/" ADD_DATE="1354273529">Example.org</A>
						<DT><A HREF="ftp://example.net/" ADD_DATE="1361299010">FTP</A>
						<DT><A HREF="https://example.org/#test" ADD_DATE="1354273529">Example.org</A>
					</DL>
				</DL><p>
			</DL><p>
			`,
			func(test *adapterTest, require *require.Assertions, f forms.Binder, data []byte) {
				require.True(f.IsValid())
				adapter := test.adapter.(importer.ImportWorker)
				err := adapter.LoadData(data)
				require.NoError(err)

				type bookmarkItem struct {
					Link       string
					Title      string
					Created    time.Time
					Labels     types.Strings
					IsArchived bool
				}
				items := []bookmarkItem{}
				for {
					item, err := adapter.Next()
					if err == io.EOF {
						break
					}
					require.NoError(err)
					bi := bookmarkItem{Link: item.URL()}
					meta, err := item.(importer.BookmarkEnhancer).Meta()
					require.NoError(err)

					bi.Title = meta.Title
					bi.Created = meta.Created
					bi.Labels = meta.Labels
					bi.IsArchived = meta.IsArchived

					items = append(items, bi)
				}

				expected := []bookmarkItem{
					{"https://example.org/", "Example.org", time.Date(2012, 11, 30, 11, 5, 29, 0, time.UTC), types.Strings{}, false},
					{"https://example.net/", "Example.net", time.Date(2013, 11, 26, 10, 38, 19, 0, time.UTC), types.Strings{}, false},
					{"http://blog.mozilla.com/", "Mozilla News", time.Date(2020, 9, 29, 20, 32, 45, 0, time.UTC), types.Strings{"mozilla", "blog", "test"}, true},
					{"https://www.mozilla.org/en-US/firefox/central/", "Getting Started", time.Date(2019, 12, 18, 7, 9, 39, 0, time.UTC), types.Strings{}, false},
				}
				require.Equal(expected, items)
			},
		},
		{
			importer.LoadAdapter("goodlinks"),
			``,
			func(_ *adapterTest, require *require.Assertions, f forms.Binder, _ []byte) {
				require.False(f.IsValid())
				require.Equal("Empty or invalid import file", f.Get("data").Errors.Error())
			},
		},
		{
			importer.LoadAdapter("goodlinks"),
			`
			[{
				"title": "Shodan",
				"url": "https:\/\/www.startpage.com\/",
				"tags": ["search"],
				"starred": false,
				"summary": "Search engine of the Internet.",
				"originalURL": "https:\/\/www.startpage.com",
				"addedAt": 1588601562
			}, {
				"title": "Home | LinuxServer.io",
				"url": "https:\/\/www.linuxserver.io\/",
				"starred": false,
				"originalURL": "https:\/\/www.linuxserver.io",
				"addedAt": 1589621418,
				"tags": ["linux", "docker"],
				"summary": "We are a group of like-minded enthusiasts from across the world who build and maintain the largest collection of Docker images on the web, and at our core are the principles behind Free and Open Source Software. Our primary goal is to provide easy-to-use and streamlined Docker images with clear and concise documentation."
			}
			]
			`,
			func(test *adapterTest, require *require.Assertions, f forms.Binder, data []byte) {
				require.True(f.IsValid())
				adapter := test.adapter.(importer.ImportWorker)
				err := adapter.LoadData(data)
				require.NoError(err)

				type bookmarkItem struct {
					Link     string
					Created  time.Time
					Labels   types.Strings
					IsMarked bool
				}
				items := []bookmarkItem{}
				for {
					item, err := adapter.Next()
					if err == io.EOF {
						break
					}
					require.NoError(err)
					bi := bookmarkItem{Link: item.URL()}
					meta, err := item.(importer.BookmarkEnhancer).Meta()
					require.NoError(err)

					bi.Created = meta.Created
					bi.Labels = meta.Labels
					bi.IsMarked = meta.IsMarked

					items = append(items, bi)
				}

				expected := []bookmarkItem{
					{"https://www.startpage.com/", time.Date(2020, time.May, 4, 14, 12, 42, 0, time.UTC), types.Strings{"search"}, false},
					{"https://www.linuxserver.io/", time.Date(2020, time.May, 16, 9, 30, 18, 0, time.UTC), types.Strings{"linux", "docker"}, false},
				}
				require.Equal(expected, items)
			},
		},
		{
			importer.LoadAdapter("pocket-file"),
			``,
			func(_ *adapterTest, require *require.Assertions, f forms.Binder, _ []byte) {
				require.False(f.IsValid())
				require.Equal("Empty or invalid import file", f.Get("data").Errors.Error())
			},
		},
		{
			importer.LoadAdapter("pocket-file"),
			`
			<!DOCTYPE html>
			<html>
				<!--So long and thanks for all the fish-->
				<head>
					<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
					<title>Pocket Export</title>
				</head>
				<body>
					<h1>Unread</h1>
					<ul>
						<li><a href="https://example.net/" time_added="1684913522" tags="">Example.net</a></li>
			<li><a href="https://example.org/#test" time_added="1684913346" tags="tag1,tag2">Example.net</a></li>
			<li><a href="ftp://example.net/" time_added="1684913346" tags="tag2"></a></li>
			<li><a href="https://example.net/#foo" time_added="1684913522" tags="">Example.net</a></li>
					</ul>
					<h1>Read Archive</h1>
					<ul>
						<li><a href="https://example.org/read" time_added="1712037544" tags="">Read article</a></li>
					</ul>
				</body>
			</html>
			`,
			func(test *adapterTest, require *require.Assertions, f forms.Binder, data []byte) {
				require.True(f.IsValid())
				adapter := test.adapter.(importer.ImportWorker)
				err := adapter.LoadData(data)
				require.NoError(err)

				type bookmarkItem struct {
					Link       string
					Title      string
					Created    time.Time
					Labels     types.Strings
					IsArchived bool
				}
				items := []bookmarkItem{}
				for {
					item, err := adapter.Next()
					if err == io.EOF {
						break
					}
					require.NoError(err)
					bi := bookmarkItem{Link: item.URL()}
					meta, err := item.(importer.BookmarkEnhancer).Meta()
					require.NoError(err)

					bi.Title = meta.Title
					bi.Created = meta.Created
					bi.Labels = meta.Labels
					bi.IsArchived = meta.IsArchived

					items = append(items, bi)
				}

				expected := []bookmarkItem{
					{"https://example.org/", "Example.net", time.Date(2023, time.May, 24, 7, 29, 6, 0, time.UTC), types.Strings{"tag1", "tag2"}, false},
					{"https://example.net/", "Example.net", time.Date(2023, time.May, 24, 7, 32, 2, 0, time.UTC), types.Strings{}, false},
					{"https://example.org/read", "Read article", time.Date(2024, time.April, 2, 5, 59, 4, 0, time.UTC), types.Strings{}, true},
				}
				require.Equal(expected, items)
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			f := importer.NewImportForm(test.adapter)

			f.Get("data").Set(strings.NewReader(test.data))
			f.Bind()
			data, err := test.adapter.Params(f)
			require.NoError(t, err)
			test.assert(&test, require.New(t), f, data)
		})
	}
}

func TestWallabagImporter(t *testing.T) {
	adapter := importer.LoadAdapter("wallabag")
	f := importer.NewImportForm(adapter)
	f.Get("url").Set("https://wallabag/")
	f.Get("username").Set("user")
	f.Get("password").Set("pass")
	f.Get("client_id").Set("client_id")
	f.Get("client_secret").Set("client_secret")
	f.Bind()

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "/oauth/v2/token", httpmock.NewJsonResponderOrPanic(
		http.StatusOK,
		map[string]string{
			"access_token": "1234",
		},
	))

	httpmock.RegisterRegexpResponder("GET", regexp.MustCompile(`^/api/entries\?`), func(r *http.Request) (*http.Response, error) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))

		var next map[string]string
		if page < 5 {
			q := r.URL.Query()
			q.Set("page", strconv.Itoa(page+1))
			r.URL.RawQuery = q.Encode()
			next = map[string]string{
				"href": r.URL.String(),
			}
		}

		response := map[string]any{
			"_links": map[string]any{
				"next": next,
			},
		}

		items := []map[string]any{}
		for _, x := range []string{"a", "b", "c"} {
			items = append(items, map[string]any{
				"is_archived":     0,
				"is_starred":      0,
				"title":           fmt.Sprintf("Article %d/%s", page, x),
				"url":             fmt.Sprintf("https://example.net/%d/article-%s", page, x),
				"content":         fmt.Sprintf("<p>some content %d - %s</p>", page, x),
				"created_at":      "2024-01-02 12:23:43",
				"published_at":    "2022-01-02 12:23:43",
				"published_by":    []string{},
				"language":        "en",
				"tags":            []string{},
				"preview_picture": fmt.Sprintf("https://example.net/picture-%d%s.webp", page, x),
				"headers":         map[string]string{},
			})
		}
		response["_embedded"] = map[string]any{
			"items": items,
		}

		return httpmock.NewJsonResponse(200, response)
	})

	require := require.New(t)

	data, err := adapter.Params(f)
	require.NoError(err)
	require.True(f.IsValid())
	require.Equal(`{"url":"https://wallabag","token":"1234"}`, string(data))

	worker := adapter.(importer.ImportWorker)
	err = worker.LoadData(data)
	require.NoError(err)

	i := 0
	letters := []string{"a", "b", "c"}
	for {
		item, err := worker.Next()
		if err == io.EOF {
			break
		}
		require.NoError(err)

		page := 1 + i/3
		x := letters[i%3]
		i++

		require.Equal(fmt.Sprintf("https://example.net/%d/article-%s", page, x), item.URL())
		bi, err := item.(importer.BookmarkEnhancer).Meta()
		require.NoError(err)

		require.Equal(fmt.Sprintf("Article %d/%s", page, x), bi.Title)
		require.Equal(time.Date(2024, time.January, 2, 12, 23, 43, 0, time.UTC), bi.Created)
		require.Equal(time.Date(2022, time.January, 2, 12, 23, 43, 0, time.UTC), bi.Published)

		resources := item.(importer.BookmarkResourceProvider).Resources()
		require.Len(resources, 1)

		require.Equal(
			fmt.Sprintf(
				`<html><head><meta property="og:image" content="https://example.net/picture-%d%s.webp"/></head><body><p>some content %d - %s</p></body></html>`,
				page, x, page, x,
			),
			string(resources[0].Data),
		)
	}
}

func TestOmnivoreImporter(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(
		"POST",
		"/api/graphql",
		func(r *http.Request) (*http.Response, error) {
			token := r.Header.Get("Authorization")

			var payload struct {
				Query         string         `json:"query"`
				OperationName string         `json:"operationName"`
				Variables     map[string]any `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				return nil, err
			}

			switch {
			case payload.OperationName == "" && strings.HasPrefix(payload.Query, "query Viewer{me"):
				if token == "failed" {
					return httpmock.NewJsonResponse(500, map[string]any{
						"errors": []any{},
					})
				}
				return httpmock.NewJsonResponse(200, map[string]any{
					"data": map[string]any{
						"me": map[string]any{
							"id":   "abc",
							"name": "alice",
						},
					},
				})
			case payload.OperationName == "Search":
				after, _ := strconv.Atoi(payload.Variables["after"].(string))
				first := int(payload.Variables["first"].(float64))

				items := []map[string]any{}
				for x := range 25 {
					node := map[string]any{
						"id":          strconv.Itoa(after + x),
						"title":       fmt.Sprintf("Article %d", after+x),
						"url":         fmt.Sprintf("https://example.net/article-%d", after+x),
						"createdAt":   "2024-01-02 12:23:43",
						"publishedAt": "2022-01-02 12:23:43",
						"content":     fmt.Sprintf("<p>Some content %d</p>", after+x),
						"pageType":    "ARTICLE",
						"author":      "",
						"image":       fmt.Sprintf("https://example.net/picture-%d.webp", after+x),
						"siteIcon":    "https://example.net/icon.png",
						"description": fmt.Sprintf("Description %d", after+x),
						"labels":      []any{},
						"state":       "SUCCEEDED",
					}
					if after+x == 0 {
						node["author"] = "Someone"
						node["state"] = "ARCHIVED"
						node["labels"] = []map[string]string{
							{"name": "label 1"}, {"name": "label 2"},
						}
					}

					items = append(items, map[string]any{
						"cursor": strconv.Itoa(after + first),
						"node":   node,
					})
				}
				response := map[string]any{
					"data": map[string]any{
						"search": map[string]any{
							"edges": items,
							"pageInfo": map[string]any{
								"hasNextPage": after < 60,
								"startCursor": strconv.Itoa(after),
								"endCursor":   strconv.Itoa(after + first),
							},
						},
					},
				}

				return httpmock.NewJsonResponse(200, response)
			}

			return httpmock.NewJsonResponse(200, nil)
		},
	)

	t.Run("auth failed", func(t *testing.T) {
		adapter := importer.LoadAdapter("omnivore")
		f := importer.NewImportForm(adapter)
		f.Get("url").Set("https://omnivore.app/")
		f.Get("token").Set("failed")
		f.Bind()

		require := require.New(t)

		_, err := adapter.Params(f)
		require.NoError(err)
		require.False(f.IsValid())
		require.Equal("Invalid API Key", f.Get("token").Errors.Error())
	})

	t.Run("auth ok", func(t *testing.T) {
		adapter := importer.LoadAdapter("omnivore")
		f := importer.NewImportForm(adapter)
		f.Get("url").Set("https://omnivore.app/")
		f.Get("token").Set("abcd")
		f.Bind()

		require := require.New(t)

		_, err := adapter.Params(f)
		require.NoError(err)
		require.True(f.IsValid())
	})

	t.Run("import", func(t *testing.T) {
		adapter := importer.LoadAdapter("omnivore")
		f := importer.NewImportForm(adapter)
		f.Get("url").Set("https://omnivore.app/")
		f.Get("token").Set("abcd")
		f.Bind()

		require := require.New(t)

		data, err := adapter.Params(f)
		require.NoError(err)
		require.True(f.IsValid())

		worker := adapter.(importer.ImportWorker)
		err = worker.LoadData(data)
		require.NoError(err)

		i := -1
		for {
			i++
			item, err := worker.Next()
			if err == io.EOF {
				break
			}
			require.NoError(err)

			require.Equal(fmt.Sprintf("https://example.net/article-%d", i), item.URL())
			bi, err := item.(importer.BookmarkEnhancer).Meta()
			require.NoError(err)

			require.Equal(fmt.Sprintf("Article %d", i), bi.Title)
			require.Equal(fmt.Sprintf("Description %d", i), bi.Description)
			require.Equal(time.Date(2024, time.January, 2, 12, 23, 43, 0, time.UTC), bi.Created)
			require.Equal(time.Date(2022, time.January, 2, 12, 23, 43, 0, time.UTC), bi.Published)

			if i == 0 {
				fmt.Printf(">>>>\n%#v\n", bi)
				require.Equal(types.Strings{"Someone"}, bi.Authors)
				require.Equal(types.Strings{"label 1", "label 2"}, bi.Labels)
				require.True(bi.IsArchived)
			} else {
				require.False(bi.IsArchived)
			}

			resources := item.(importer.BookmarkResourceProvider).Resources()
			require.Len(resources, 1)
			require.Equal(
				fmt.Sprintf(
					`<html><head><meta property="og:image" content="https://example.net/picture-%d.webp"/><link rel="icon" href="https://example.net/icon.png"/></head><body><p>Some content %d</p></body></html>`,
					i, i,
				),
				string(resources[0].Data),
			)
		}
	})
}
