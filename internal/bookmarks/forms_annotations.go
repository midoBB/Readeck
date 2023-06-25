package bookmarks

import (
	"strings"
	"time"

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

func (f *annotationForm) addToBookmark(bi *bookmarkItem) (*BookmarkAnnotation, error) {
	annotation := &BookmarkAnnotation{
		ID:            shortuuid.New(),
		StartSelector: f.Get("start_selector").String(),
		StartOffset:   f.Get("start_offset").Value().(int),
		EndSelector:   f.Get("end_selector").String(),
		EndOffset:     f.Get("end_offset").Value().(int),
		Created:       time.Now(),
	}

	// Try to insert the new annotation
	reader, err := bi.getArticle()
	if err != nil {
		return nil, err
	}

	var doc *html.Node
	if doc, err = html.Parse(reader); err != nil {
		return nil, err
	}
	root := dom.QuerySelector(doc, "body")

	// Add annotation and store its text content
	contents := &strings.Builder{}
	err = annotation.addToNode(root, bi.annotationTag, func(n *html.Node, index int) {
		contents.WriteString(n.FirstChild.Data)
		bi.annotationCallback(annotation.ID, n, index)
	})
	if err != nil {
		return nil, err
	}

	annotation.Text = strings.TrimSpace(contents.String())

	// All good? Create the annotation now
	b := bi.Bookmark
	if b.Annotations == nil {
		b.Annotations = BookmarkAnnotations{}
	}

	b.Annotations.add(annotation)
	b.Annotations.sort(root, bi.annotationTag)

	err = b.Update(map[string]interface{}{
		"annotations": b.Annotations,
	})
	if err != nil {
		return nil, err
	}

	return annotation, nil
}
