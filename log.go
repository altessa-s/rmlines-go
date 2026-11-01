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

func (l LogLevel) String() string {
	if l == LogError {
		return "error"
	}
	return "debug"
}

// logHandler is what the C trampoline calls through. Read on whichever thread
// is parsing, swapped by SetLogger, hence atomic.
var logHandler atomic.Pointer[func(LogLevel, string)]

// logInstall serialises SetLogger, so that the C sink is installed once per
// switch from silence to a handler.
var logInstall sync.Mutex

// SetLogger names the function the parser's diagnostics go to, for the whole
// process; nil goes back to silence, which is the default. The function is
// called on whichever goroutine is parsing, so one that touches shared state
// must guard it itself.
//
// Nothing in a message is a secret: every value is a count, a block type or a
// raw style number, never the page's writing, text or file names.
func SetLogger(handler func(level LogLevel, message string)) {
	logInstall.Lock()
	defer logInstall.Unlock()
	if handler == nil {
		C.rm_log_silence()
		logHandler.Store(nil)
		return
	}
	// The C library keeps every sink it was ever given, so a replacement
	// handler swaps here instead of installing a sink anew.
	if logHandler.Swap(&handler) == nil {
		C.rmgo_log_install()
	}
}

//export rmgoLog
func rmgoLog(level C.int, message *C.char) {
	if handler := logHandler.Load(); handler != nil {
		(*handler)(LogLevel(level), C.GoString(message))
	}
}
