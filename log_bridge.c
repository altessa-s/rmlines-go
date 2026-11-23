// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// The C side of SetLogger: a sink with the signature the library calls, which
// hands each line to the exported Go function. Kept apart from log.go because
// a Go file that exports to C may only declare, not define, in its preamble.

#include "_cgo_export.h"
#include "rmlines/log.h"

static _Thread_local uintptr_t session_buffer;

void rmgo_session_log_buffer(uintptr_t handle) {
    session_buffer = handle;
}

static void rmgo_log_sink(rm_log_level level, const char *message, void *context) {
    (void)context;
    rmgoLog((int)level, (char *)message, session_buffer);
}

void rmgo_log_install(void) {
    rm_log_install(rmgo_log_sink, nullptr);
}
