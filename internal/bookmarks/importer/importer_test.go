// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer_test

import (
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"codeberg.org/readeck/readeck/internal/bookmarks/importer"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/forms"
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
					<DT><A HREF="http://blog.mozilla.com/" ADD_DATE="1601411565">Mozilla News</A>
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
					Link    string
					Title   string
					Created time.Time
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

					items = append(items, bi)
				}

				expected := []bookmarkItem{
					{"https://example.org/", "Example.org", time.Date(2012, 11, 30, 11, 5, 29, 0, time.UTC)},
					{"https://example.net/", "Example.net", time.Date(2013, 11, 26, 10, 38, 19, 0, time.UTC)},
					{"http://blog.mozilla.com/", "Mozilla News", time.Date(2020, 9, 29, 20, 32, 45, 0, time.UTC)},
					{"https://www.mozilla.org/en-US/firefox/central/", "Getting Started", time.Date(2019, 12, 18, 7, 9, 39, 0, time.UTC)},
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
			f := test.adapter.Form()
			f.Get("data").Set(strings.NewReader(test.data))
			f.Bind()
			data, err := test.adapter.Params(f)
			require.NoError(t, err)
			test.assert(&test, require.New(t), f, data)
		})
	}
}
