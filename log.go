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
// The handler runs on a separate goroutine, synchronously with the parse
// that produced the line. It does not inherit goroutine-bound state such as
// runtime.LockOSThread; shared state still needs its own guard. A panic or
// runtime.Goexit is contained and loses only that line, so the parser can
// finish and release its memory.
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
	text := C.GoString(message)
	_, _ = runCallback(func() { (*handler)(LogLevel(level), text) })
}

// runCallback waits for fn on a separate goroutine so neither a panic nor
// Goexit can unwind the C frames of the goroutine waiting for it.
func runCallback(fn func()) (returned bool, panicValue any) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { panicValue = recover() }()
		fn()
		returned = true
	}()
	<-done
	return returned, panicValue
}
