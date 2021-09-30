package bookmarks

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/doug-martin/goqu/v9"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/readeck/readeck/pkg/form"
)

// filterForm is the form used by any bookmark filtering operation.
// From simple label filter to complex search.
type filterForm struct {
	// This lets us override some Form methods.
	*form.Form `json:"-"`

	Search       string     `json:"q"`
	Title        string     `json:"title"`
	Author       string     `json:"author"`
	Site         string     `json:"site"`
	Type         typeChoice `json:"type"`
	Labels       string     `json:"label"`
	IsMarked     *bool      `json:"is_marked"`
	IsArchived   *bool      `json:"is_archived"`
	IDs          []string   `json:"id"`
	noPagination bool
	st           searchString
}

// typeChoice is a form field with a choice of bookmark types.
type typeChoice string

// Options returns the field's choices, including an empty value.
func (t typeChoice) Options() [][2]string {
	return append([][2]string{{"", ""}}, availableTypes...)
}

// Validate performs the field validation.
func (t typeChoice) Validate(f *form.Field) error {
	value, _ := validation.Indirect(f.Value())
	str, err := validation.EnsureString(value)
	if err != nil {
		return err
	}

	if str == "" {
		return nil
	}

	if _, ok := AvailableTypes()[str]; ok {
		return nil
	}

	return fmt.Errorf("must be one of %s", strings.Join(ValidTypes(), ", "))
}

// newFilterForm creates a new instance of filterForm.
func newFilterForm() *filterForm {
	res := &filterForm{st: searchString{}}
	res.Form = form.NewForm(res)
	return res
}

// newContextFilterForm returns an instance of filterForm. If one already
// exists in the given context, it's reused, otherwise it returns a new one.
func newContextFilterForm(c context.Context) *filterForm {
	ff, ok := c.Value(ctxFiltersKey{}).(*filterForm)
	if !ok {
		ff = newFilterForm()
	}

	return ff
}

// BindValues reads a query string values and adds them to the
// filterForm properties.
func (ff *filterForm) BindValues(values url.Values) {
	// Tri-values (null, true, false) don't play well with the
	// decoder, so let's just remove them when they're empty.
	if values.Get("is_marked") == "" {
		values.Del("is_marked")
	}
	if values.Get("is_archived") == "" {
		values.Del("is_archived")
	}

	ff.Form.BindValues(values)

	// First, we must build a search string based on
	// the provided free form search and
	// what we might have in the following fields:
	// title, author, site, label
	if ff.Search != "" {
		ff.st = newSearchString(ff.Search)
	}

	for _, x := range [][2]string{
		{"title", ff.Title},
		{"author", ff.Author},
		{"site", ff.Site},
		{"label", ff.Labels},
	} {
		ff.st.addField(x[0], x[1])
	}

	// Remove any duplicate in the final search string
	ff.st = ff.st.dedup()

	// Update the specific search fields
	ff.Search = ff.st.fieldString("")
	ff.Title = ff.st.fieldString("title")
	ff.Author = ff.st.fieldString("author")
	ff.Site = ff.st.fieldString("site")
	ff.Labels = ff.st.fieldString("label")
}

// setMarked sets the IsMarked property.
func (ff *filterForm) setMarked(v bool) {
	ff.IsMarked = &v
}

// setArchived sets the IsArchived property.
func (ff *filterForm) setArchived(v bool) {
	ff.IsArchived = &v
}

// saveContext returns a context containing this filterForm.
// It can be retrieved using newContextFilterForm().
func (ff *filterForm) saveContext(c context.Context) context.Context {
	return context.WithValue(c, ctxFiltersKey{}, ff)
}

// toSelectDataSet returns an augmented select dataset including all the filter
// clauses.
func (ff *filterForm) toSelectDataSet(ds *goqu.SelectDataset) *goqu.SelectDataset {
	if ff.IsMarked != nil {
		ds = ds.Where(goqu.C("is_marked").Table("b").Eq(goqu.V(ff.IsMarked)))
	}
	if ff.IsArchived != nil {
		ds = ds.Where(goqu.C("is_archived").Table("b").Eq(goqu.V(ff.IsArchived)))
	}
	if ff.Type != "" {
		ds = ds.Where(goqu.C("type").Table("b").Eq(ff.Type))
	}

	// Separate labels from the final search string
	var labels searchString
	var search searchString
	if len(ff.st) > 0 {
		labels, search = ff.st.popField("label")
	}

	// Label filter
	if len(labels) > 0 {
		l := make([]string, len(labels))
		for i, x := range labels {
			l[i] = x.Value
		}
		ds = Bookmarks.AddLabelFilter(ds, l)
	}

	// Search string
	if len(search) > 0 {
		ds = search.toSelectDataSet(ds)
	}

	// Filtering by ids. In this case we include all the given IDs and we sort the
	// result according to the IDs order.
	if len(ff.IDs) > 0 {
		ds = ds.Where(
			goqu.C("uid").Table("b").In(ff.IDs),
		)

		orderging := goqu.Case().Value(goqu.C("uid").Table("b"))
		for i, x := range ff.IDs {
			orderging = orderging.When(x, i)
		}
		ds = ds.Order(orderging.Asc())
	}

	return ds
}
