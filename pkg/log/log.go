/*
Copyright (c) 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package log provides structured, leveled diagnostic logging for core
// layer workflows, propagated through context.Context rather than passed
// as a bespoke CLI-specific parameter. Unlike pkg/reporter, it carries no
// CLI presentation concerns (no color, no terminal detection, no display
// strings), so it is safe for pkg/ and internal/core/ code to import. See
// guidelines/structured-logging.md for the full design and the relationship
// to reporter output.
package log

import (
	"context"
	"io"
	"log/slog"
)

// New returns a structured logger that writes to w at the given level.
// The CLI layer is responsible for choosing w and level; this package has
// no opinion on either and does not read any global or CLI-layer state.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

// ctxKey is unexported so only this package can set or retrieve the
// logger stored on a context.Context.
type ctxKey struct{}

// discard is returned by FromContext when no logger was attached, so
// callers never need a nil check and never need to wire up logging just to
// call a workflow function.
var discard = slog.New(slog.DiscardHandler)

// WithLogger returns a copy of ctx carrying logger, retrievable with
// FromContext.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext returns the logger attached to ctx with WithLogger, or a
// discard logger if none was attached or a nil logger was attached.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return discard
}
