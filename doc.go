// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// Package rmlines reads and writes the reMarkable .rm v6 page format: the
// page and its strokes, typed text and its paragraphs, highlights and images,
// and fresh pages the tablet accepts. Nothing here draws.
//
// The package binds rmlines-lib, a C23 library vendored into this module and
// compiled by cgo; building it needs a C23 compiler (clang 19 or newer, GCC 13
// or newer) and nothing else. Every value the API returns is a Go value: the
// C memory behind a call is released before the call returns, and nothing the
// caller holds points into it.
//
// # Reading
//
// [Parse] reads one page. A block that will not read costs that block, not the
// page: it is stepped over and counted in [Page.SkippedBlocks], and a line
// goes to the logger installed with [SetLogger].
//
// # Writing
//
// [Write] turns a [Draft] into the bytes of a fresh page. It never edits an
// existing file; that is [EraseItems], which erases strokes and highlights in
// place, the way the tablet's own eraser does.
//
// # Errors
//
// Every failure is an [*Error]. Its [Status] says which, and the sentinel
// values ([ErrUnsupportedVersion], [ErrItemsNotFound], …) match it through
// [errors.Is].
// A count or a length the C library states and a Go int cannot hold is
// [ErrOutOfMemory]: no Go value of that size can be allocated.
//
// # Concurrency
//
// Parse, Write, EraseItems and HighlighterItems keep no state between calls:
// each call allocates and frees its own memory, so any number may run at once.
// The logger is the one process-wide setting, and it may be replaced while
// calls are running.
package rmlines
