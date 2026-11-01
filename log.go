// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

/*
#include "rmlines/log.h"

void rmgo_log_install(void);
*/
import "C"

import (
	"sync"
	"sync/atomic"
)

// LogLevel is how much a diagnostic line matters: a piece of the page was
// lost, or the format held something the parser does not know and read past.
type LogLevel int

// The log levels.
const (
	LogDebug LogLevel = iota
	LogError
)

// String answers "error" for LogError and "debug" for every other level.
func (l LogLevel) String() string {
	if l == LogError {
		return "error"
	}
	return "debug"
}

// logHandler is what the C sink calls through. Read on whichever thread is
// parsing and swapped by SetLogger, hence atomic; nil drops every line.
var logHandler atomic.Pointer[func(LogLevel, string)]

// logInstall installs the C sink once per process. The C library keeps every
// sink it is ever given, so installing on every SetLogger would grow without
// bound; the sink stays installed and the Go handler decides.
var logInstall sync.Once

// logInstalls counts installations of the C sink, for the tests.
var logInstalls atomic.Int32

// SetLogger names the function the parser's diagnostics go to, for the whole
// process; nil goes back to silence, which is the default. It may be called
// at any time, from any goroutine.
//
// The handler is called on whichever goroutine is parsing, so one that
// touches shared state must guard it itself. It must not panic: a panic is
// recovered and discarded with its line, because unwinding through the C
// parser would skip the release of its memory.
//
// Once a logger has been set, the C library formats its lines even while the
// handler is nil; a line is a few dozen bytes, built only where something was
// lost or unknown.
//
// Nothing in a message is a secret: every value is a count, a block type or a
// raw style number, never the page's writing, text or file names.
func SetLogger(handler func(level LogLevel, message string)) {
	if handler == nil {
		logHandler.Store(nil)
		return
	}
	logHandler.Store(&handler)
	logInstall.Do(func() {
		logInstalls.Add(1)
		C.rmgo_log_install()
	})
}

//export rmgoLog
func rmgoLog(level C.int, message *C.char) {
	handler := logHandler.Load()
	if handler == nil {
		return
	}
	defer func() {
		_ = recover() // the line is dropped; the parse goes on
	}()
	(*handler)(LogLevel(level), C.GoString(message))
}
