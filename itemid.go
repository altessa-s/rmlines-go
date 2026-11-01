// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

/*
#include <stdlib.h>
#include "rmlines/rmlines.h"
*/
import "C"

import (
	"strconv"
	"strings"
	"unsafe"
)

// ItemID is what a page calls one item of its scene, a stroke or a highlight:
// the author of the edit and its counter. It is the handle an edit finds the
// item by, and by nothing about its looks.
type ItemID struct {
	Author  uint8
	Counter uint64
}

// MaxItemIDLength is how many bytes the spelling of the widest identifier
// takes.
const MaxItemIDLength = 24

func (id ItemID) toC() C.rm_item_id {
	return C.rm_item_id{author: C.uint8_t(id.Author), counter: C.uint64_t(id.Counter)}
}

// String spells the identifier as author:counter.
func (id ItemID) String() string {
	var buf [MaxItemIDLength + 1]C.char
	n := C.rm_item_id_format(id.toC(), &buf[0], C.size_t(len(buf)))
	return C.GoStringN(&buf[0], n)
}

// Equal reports whether two identifiers name the same item.
func (id ItemID) Equal(other ItemID) bool {
	return bool(C.rm_item_id_equal(id.toC(), other.toC()))
}

// ParseItemID reads an identifier out of its author:counter spelling. Each
// number takes one leading sign, a plus on anything and a minus only on a
// nought; anything else — an author past 255, a counter past 64 bits, a space,
// a third part — is refused with an error matching [ErrInvalidArgument].
func ParseItemID(s string) (ItemID, error) {
	// A NUL would end the C string early and leave its tail unread.
	if strings.IndexByte(s, 0) < 0 {
		text := C.CString(s)
		defer C.free(unsafe.Pointer(text))
		var out C.rm_item_id
		if C.rm_item_id_parse(text, &out) {
			return itemIDFromC(out), nil
		}
	}
	return ItemID{}, &Error{
		Status:  StatusInvalidArgument,
		Message: "not an item identifier: " + strconv.Quote(s),
	}
}
