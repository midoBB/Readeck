// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package converter provides bookmark export/converter tooling.
package converter

import (
	"context"
	"io"
	"net/http"

	"codeberg.org/readeck/readeck/internal/bookmarks"
)

type contextKey struct {
	name string
}

var (
	ctxURLReplaceKey         = &contextKey{"baseURL"}
	ctxAnnotationTagKey      = &contextKey{"annotationTag"}
	ctxAnnotationCallbackKey = &contextKey{"annotationCallback"}
)

// Exporter describes a bookmarks exporter.
type Exporter interface {
	Export(ctx context.Context, w io.Writer, r *http.Request, bookmarks []*bookmarks.Bookmark) error
}

// WithURLReplacer adds to context the URL replacment values for image sources.
func WithURLReplacer(ctx context.Context, orig, repl string) context.Context {
	return context.WithValue(ctx, ctxURLReplaceKey, [2]string{orig, repl})
}

func getURLReplacer(ctx context.Context) (orig, repl string, ok bool) {
	s, ok := ctx.Value(ctxURLReplaceKey).([2]string)
	return s[0], s[1], ok
}

// WithAnnotationTag adds to context the annotation tag and callback function.
func WithAnnotationTag(ctx context.Context, tag string, callback annotationCallback) context.Context {
	ctx = context.WithValue(ctx, ctxAnnotationTagKey, tag)
	ctx = context.WithValue(ctx, ctxAnnotationCallbackKey, callback)
	return ctx
}

func getAnnotationTag(ctx context.Context) (tag string, callback annotationCallback) {
	tag, _ = ctx.Value(ctxAnnotationTagKey).(string)
	callback, _ = ctx.Value(ctxAnnotationCallbackKey).(annotationCallback)
	return
}
