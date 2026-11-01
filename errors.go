// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

/*
#include "rmlines/rmlines.h"
*/
import "C"

import (
	"strconv"
	"strings"
)

// Status says what went wrong. The numbers are the C library's.
type Status int

// The statuses a call can answer with.
const (
	StatusOK Status = iota
	// StatusUnexpectedEOF: the stream is shorter than a header, or a block
	// declares a length past the end of the file.
	StatusUnexpectedEOF
	// StatusUnsupportedFormat: the bytes do not begin the way a page does.
	// [Error.Message] holds the first of them.
	StatusUnsupportedFormat
	// StatusUnsupportedVersion: a page of a generation this reader does not
	// parse. [Error.Version] is the number in its header.
	StatusUnsupportedVersion
	// StatusMalformedPage: a block that would not read. The parser counts it
	// in [Page.SkippedBlocks] rather than failing the page.
	StatusMalformedPage
	StatusOutOfMemory
	// StatusAnchorOutOfRange: a mark or an image of a draft is fastened to a
	// paragraph the draft does not have. [Error.Value] names it.
	StatusAnchorOutOfRange
	// StatusTooManyToNumber: more items of one kind than a page's identifier
	// ranges can name. [Error.Value] is the limit, [Error.Message] the kind.
	StatusTooManyToNumber
	// StatusInvalidArgument: an argument the call cannot be made with.
	StatusInvalidArgument
	// StatusItemsNotFound: an identifier asked for is not a living stroke or
	// highlight of the page. [Error.Value] is how many were missing.
	StatusItemsNotFound
	// StatusUnverified: an edited page did not read back as the edit meant.
	StatusUnverified
	// StatusIncompletePage: blocks of the page would not read, so what it
	// holds cannot be told in full. [Error.Value] is how many.
	StatusIncompletePage
)

// String answers the status as a bare English phrase.
func (s Status) String() string {
	return C.GoString(C.rm_status_string(C.rm_status(s)))
}

// Error is what every call of this package fails with. Every field but Status
// is only as meaningful as Status says.
type Error struct {
	Status Status
	// Version is the generation a header named, for StatusUnsupportedVersion.
	Version int
	// Value is the paragraph, the limit or the count the status speaks of.
	Value int64
	// Message is what the C library had to say: the first bytes of the
	// header for StatusUnsupportedFormat, otherwise a phrase.
	Message string
}

// Sentinels for [errors.Is]: an *Error matches the sentinel of its Status,
// whatever its other fields hold.
var (
	ErrUnexpectedEOF      = &Error{Status: StatusUnexpectedEOF}
	ErrUnsupportedFormat  = &Error{Status: StatusUnsupportedFormat}
	ErrUnsupportedVersion = &Error{Status: StatusUnsupportedVersion}
	ErrMalformedPage      = &Error{Status: StatusMalformedPage}
	ErrOutOfMemory        = &Error{Status: StatusOutOfMemory}
	ErrAnchorOutOfRange   = &Error{Status: StatusAnchorOutOfRange}
	ErrTooManyToNumber    = &Error{Status: StatusTooManyToNumber}
	ErrInvalidArgument    = &Error{Status: StatusInvalidArgument}
	ErrItemsNotFound      = &Error{Status: StatusItemsNotFound}
	ErrUnverified         = &Error{Status: StatusUnverified}
	ErrIncompletePage     = &Error{Status: StatusIncompletePage}
)

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("rmlines: ")
	b.WriteString(e.Status.String())
	switch e.Status {
	case StatusUnsupportedVersion:
		b.WriteString(" (version ")
		b.WriteString(strconv.Itoa(e.Version))
		b.WriteString(")")
	case StatusUnsupportedFormat:
		// The header's bytes may be anything; quoted, they stay one line.
		if e.Message != "" {
			b.WriteString(": ")
			b.WriteString(strconv.Quote(e.Message))
		}
		return b.String()
	}
	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	return b.String()
}

// Is reports whether target is an *Error of the same Status.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Status == e.Status
}

// errorOf turns a C status and the error record beside it into a Go error,
// nil for success.
func errorOf(status C.rm_status, cerr *C.rm_error) error {
	if status == C.RM_OK {
		return nil
	}
	return &Error{
		Status:  Status(status),
		Version: int(cerr.version),
		Value:   int64(cerr.value),
		Message: C.GoString(&cerr.message[0]),
	}
}
