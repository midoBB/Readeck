package bookmarks

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// bookmarkContainer is a wrapper around zip.ReadCloser
// to handle a bookmark's zipfile.
type bookmarkContainer struct {
	z              *zip.ReadCloser
	articleName    string
	articleContent *strings.Builder
}

// openContainer opens the bookmark's zipfile and returns a new
// bookmarkContainer instance.
func (b *Bookmark) openContainer() (*bookmarkContainer, error) {
	p := b.getFilePath()
	if p == "" {
		return nil, os.ErrNotExist
	}

	z, err := zip.OpenReader(p)
	if err != nil {
		return nil, err
	}

	res := &bookmarkContainer{
		z:              z,
		articleContent: new(strings.Builder),
	}
	if a, ok := b.Files["article"]; ok {
		res.articleName = a.Name
	}

	return res, nil
}

// Close closes the zipfile.
func (c *bookmarkContainer) Close() error {
	return c.z.Close()
}

// ListResources returns a list of files located under "_resources/".
func (c *bookmarkContainer) ListResources() []*zip.File {
	res := []*zip.File{}
	for _, entry := range c.z.File {
		if !strings.HasSuffix(entry.Name, "/") && strings.HasPrefix(entry.Name, resourceDirName) {
			res = append(res, entry)
		}
	}

	return res
}

// LoadArticle loads the bookmark´s article when it exists.
func (c *bookmarkContainer) LoadArticle() error {
	if c.articleName == "" {
		return os.ErrNotExist
	}

	fp, err := c.z.Open(c.articleName)
	if err != nil {
		return err
	}

	_, err = io.Copy(c.articleContent, fp)

	return err
}

// ReplaceLinks replaces all the link to _resources/* in the article content.
func (c *bookmarkContainer) ReplaceLinks(orig, repl string) (err error) {
	args := []string{}
	for _, x := range c.ListResources() {
		args = append(args,
			fmt.Sprintf("%s/%s", orig, path.Base(x.Name)),
			fmt.Sprintf("%s/%s", repl, path.Base(x.Name)),
		)
	}

	replacer := strings.NewReplacer(args...)
	res := replacer.Replace(c.articleContent.String())
	c.articleContent.Reset()
	_, err = c.articleContent.WriteString(res)
	return
}

// ExtractBody extract the content of the article's HTML body.
func (c *bookmarkContainer) ExtractBody() (err error) {
	res := rxHTMLStart.ReplaceAllString(c.articleContent.String(), "")
	res = rxHTMLEnd.ReplaceAllString(res, "")
	c.articleContent.Reset()
	_, err = c.articleContent.WriteString(res)
	return
}

// GetArticle returns a string of the article's HTML.
func (c *bookmarkContainer) GetArticle() string {
	return c.articleContent.String()
}

func (c *bookmarkContainer) GetFile(name string) ([]byte, error) {
	fd, err := c.z.Open(name)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(fd)
}
