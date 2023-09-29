// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts

import (
	"context"

	"github.com/sirupsen/logrus"
)

var loggerCtxKey = &contextKey{"logger"}

// SetLogger sets the runtime's log entry.
func (vm *Runtime) SetLogger(entry *logrus.Entry) {
	vm.ctx = context.WithValue(vm.ctx, loggerCtxKey, entry)
}

// GetLogger returns the runtime's log entry or a default one
// when not set.
func (vm *Runtime) GetLogger() *logrus.Entry {
	var entry *logrus.Entry
	var ok bool
	if entry, ok = vm.ctx.Value(loggerCtxKey).(*logrus.Entry); !ok {
		entry = logrus.NewEntry(logrus.StandardLogger())
	}

	// Add the script field when present
	if scriptName := vm.Get("__name__"); scriptName != nil {
		entry = entry.WithField("script", scriptName.String())
	}

	return entry
}

func (vm *Runtime) startConsole() {
	console := vm.NewObject()
	console.Set("debug", logFunc("debug", vm.GetLogger))
	console.Set("error", logFunc("error", vm.GetLogger))
	console.Set("info", logFunc("info", vm.GetLogger))
	console.Set("log", logFunc("log", vm.GetLogger))
	console.Set("warn", logFunc("warn", vm.GetLogger))

	vm.Set("console", console)
}

func logFunc(level string, getLogger func() *logrus.Entry) func(...any) {
	return func(args ...any) {
		msg := []any{}
		fields := logrus.Fields{}

		for _, x := range args {
			if f, ok := x.(map[string]any); ok {
				for k, v := range f {
					fields[k] = v
				}
			} else {
				msg = append(msg, x)
			}
		}

		switch level {
		case "debug":
			getLogger().WithFields(fields).Debug(msg...)
		case "error":
			getLogger().WithFields(fields).Error(msg...)
		case "warn":
			getLogger().WithFields(fields).Warn(msg...)
		default:
			getLogger().WithFields(fields).Info(msg...)
		}
	}
}
