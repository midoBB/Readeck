package bookmarks

import (
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/go-shiori/dom"
	"github.com/lithammer/shortuuid/v3"
	"golang.org/x/net/html"

	"github.com/readeck/readeck/pkg/forms"
)

type annotationForm struct {
	*forms.Form
}

func newAnnotationForm() *annotationForm {
	return &annotationForm{forms.Must(
		forms.NewTextField("start_selector", forms.Required, forms.Trim),
		forms.NewIntegerField("start_offset", forms.Required, forms.Gte(0)),
		forms.NewTextField("end_selector", forms.Required, forms.Trim),
		forms.NewIntegerField("end_offset", forms.Required, forms.Gte(0)),
	)}
}

func (f *annotationForm) addToBookmark(bi *bookmarkItem) (string, *BookmarkAnnotation, error) {
	id := shortuuid.New()
	annotation := &BookmarkAnnotation{
		StartSelector: f.Get("start_selector").String(),
		StartOffset:   f.Get("start_offset").Value().(int),
		EndSelector:   f.Get("end_selector").String(),
		EndOffset:     f.Get("end_offset").Value().(int),
		Created:       time.Now(),
	}

	// Try to insert the new annotation
	reader, err := bi.getArticle()
	if err != nil {
		return "", nil, err
	}

	var doc *html.Node
	if doc, err = html.Parse(reader); err != nil {
		return "", nil, err
	}
	root := dom.QuerySelector(doc, "body")

	// Add annotation and store its text content
	contents := &strings.Builder{}
	err = annotation.addToNode(root, bi.annotationTag, func(n *html.Node, index int) {
		io.WriteString(contents, n.FirstChild.Data)
	})
	if err != nil {
		return "", nil, err
	}

	annotation.Text = shortText(contents.String(), 60)

	// All good? Create the annotation now
	b := bi.Bookmark
	if b.Annotations == nil {
		b.Annotations = BookmarkAnnotations{}
	}

	b.Annotations[id] = annotation
	err = b.Update(map[string]interface{}{
		"annotations": b.Annotations,
	})
	if err != nil {
		return "", nil, err
	}

	return id, annotation, nil
}

// shortText returns a string of maxChars maximum length. It attempts to cut between words
// when possible.
func shortText(s string, maxChars int) string {
	runes := []rune(strings.TrimSpace(strings.Join(strings.Fields(s), " ")))
	if len(runes) <= maxChars {
		return string(runes)
	}

	res := &strings.Builder{}
	j := 0
	for i, word := range strings.FieldsFunc(s, unicode.IsSpace) {
		j += len(word)
		if j >= maxChars {
			if len(word) > maxChars {
				res.WriteString(word[0:maxChars])
			}
			break
		}
		if i > 0 {
			res.WriteString(" ")
		}
		res.WriteString(word)
	}
	res.WriteString("...")

	return res.String()
}
