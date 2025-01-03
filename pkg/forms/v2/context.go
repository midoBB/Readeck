// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"context"
	"time"
)

type contextKey struct {
	name string
}

type mergedCtx struct {
	base  context.Context
	extra context.Context
}

// mergeContext merges two context values. The base context still holds
// Deadline, Done and Err. Only values are looked up in the base and extra contexts.
func mergeContext(parent, main context.Context) context.Context {
	return &mergedCtx{parent, main}
}

// Deadline implements [context.Context] and returns the base's deadline.
func (ctx *mergedCtx) Deadline() (deadline time.Time, ok bool) {
	return ctx.base.Deadline()
}

// Done implements [context.Context] and returns the base's done channel.
func (ctx *mergedCtx) Done() <-chan struct{} {
	return ctx.base.Done()
}

// Err implements [context.Context] and returns the base's error.
func (ctx *mergedCtx) Err() error {
	return ctx.base.Err()
}

// Value implements [context.Context] and returns the value found in the main or base context.
func (ctx *mergedCtx) Value(key any) any {
	if value := ctx.extra.Value(key); value != nil {
		return value
	}
	return ctx.base.Value(key)
}
