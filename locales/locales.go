// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package locales provides l10n tools to render gettext text.
//
// Translations are in the "translations" embedded folder.
package locales

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path"

	"github.com/leonelquinteros/gotext"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// localesFS contains all the translation files
//
//go:embed translations/*/*.po
var localesFS embed.FS

// completionCutoff is the percentage of translated content under which
// a translation won't be loaded.
const completionCutoff = 0.9

var (
	catalog   = make(map[language.Tag]*Locale)
	allTags   = []language.Tag{}
	available [][2]string
)

// Locale combines a gotext.Translator instance for a given language
// identified by a language.Tag.
type Locale struct {
	Translator gotext.Translator
	Tag        language.Tag
}

// Gettext returns a translation.
func (t *Locale) Gettext(str string, vars ...interface{}) string {
	return t.Translator.Get(str, vars...)
}

// Ngettext returns a pluralized translation.
func (t *Locale) Ngettext(str, plural string, n int, vars ...interface{}) string {
	return t.Translator.GetN(str, plural, n, vars...)
}

// Pgettext returns a contextualized translation.
func (t *Locale) Pgettext(ctx, str string, vars ...interface{}) string {
	return t.Translator.GetC(str, ctx, vars...)
}

// Npgettext returns a pluralized contextualized translation.
func (t *Locale) Npgettext(ctx, str, plural string, n int, vars ...interface{}) string {
	return t.Translator.GetNC(str, plural, n, ctx, vars...)
}

// LoadTranslation loads the best match translation for a given locale code.
func LoadTranslation(lang string) *Locale {
	_, i := language.MatchStrings(language.NewMatcher(allTags), lang, "en-US")

	return catalog[allTags[i]]
}

// Load loads all the available translations.
func Load() {
	files, err := fs.Glob(localFilesFS(), "*/messages.po")
	if err != nil {
		panic(err)
	}

	for _, filename := range files {

		tag := language.Make(path.Dir(filename))
		var r io.Reader
		if r, err = localFilesFS().Open(filename); err != nil {
			panic(err)
		}

		if err = addLocale(tag, r); err != nil {
			panic(err)
		}
	}

	available = make([][2]string, len(allTags))
	for i, t := range allTags {
		n, _ := t.MarshalText()
		available[i] = [2]string{string(n), display.Self.Name(t)}
	}
}

// Available returns the available locales as a list of pair
// containing the langage code and its localized name.
func Available() [][2]string {
	return available
}

func localFilesFS() fs.FS {
	sub, err := fs.Sub(localesFS, "translations")
	if err != nil {
		panic(err)
	}
	return sub
}

func addLocale(tag language.Tag, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	mo := gotext.NewPo()
	mo.Parse(b)

	total := 0
	translated := 0
	for _, x := range mo.GetDomain().GetTranslations() {
		total++
		if x.IsTranslated() {
			translated++
		}
	}
	pct := float64(translated) / float64(total)

	log := slog.With(
		slog.String("tag", tag.String()),
		slog.Int("strings", total),
		slog.Int("translated", translated),
		slog.String("completion", fmt.Sprintf("%.0f%%", pct*100)),
	)

	if tag.String() != "en-US" && pct <= completionCutoff {
		log.Debug("translation not loaded", slog.Any("err", errors.New("partial")))
		return nil
	}

	log.Debug("locale loaded")

	catalog[tag] = &Locale{
		Translator: mo,
		Tag:        tag,
	}

	allTags = append(allTags, tag)
	return nil
}
