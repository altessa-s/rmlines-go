// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

/*
#include "rmlines/rmlines.h"
*/
import "C"

import "unsafe"

// EraseItems answers the page with these strokes and highlights erased, the
// way the tablet's own eraser does it: each item stays as a tombstone, with
// its identifier, parent and neighbours, and every other byte goes out as it
// came in.
//
// Every identifier must name a living stroke or highlight of the page, or
// nothing is erased: an error matching [ErrItemsNotFound]. The result is read
// back before it is answered; [ErrUnverified] if it does not read as meant.
// No identifiers at all is no edit, and the bytes come back as they went in.
// A page that does not parse is refused with the error [Parse] gives it.
func EraseItems(data []byte, ids []ItemID) ([]byte, error) {
	// The identifiers hold no pointers, so a Go slice of them may be handed
	// to C as it is.
	cids := make([]C.rm_item_id, len(ids))
	for i, id := range ids {
		cids[i] = id.toC()
	}
	var idsPointer *C.rm_item_id
	if len(cids) > 0 {
		idsPointer = &cids[0]
	}
	var out *C.uint8_t
	var size C.size_t
	var cerr C.rm_error
	status := C.rm_erase_items(bytesPointer(data), C.size_t(len(data)), idsPointer, C.size_t(len(cids)),
		&out, &size, &cerr)
	if err := errorOf(status, &cerr); err != nil {
		return nil, err
	}
	return takeBytes(out, size)
}

// HighlighterItems answers every living mark of the highlighter on a page: its
// strokes on every layer, a hidden one included, and its highlights over a
// text layer — what an erase of everything the marker did takes. Each
// identifier once, strokes in file order and then highlights.
//
// A page with a block that would not read is refused rather than answered in
// part, with an error matching [ErrIncompletePage]: the block lost may be a
// mark.
func HighlighterItems(data []byte) ([]ItemID, error) {
	var ids *C.rm_item_id
	var count C.size_t
	var cerr C.rm_error
	status := C.rm_highlighter_items(bytesPointer(data), C.size_t(len(data)), &ids, &count, &cerr)
	if err := errorOf(status, &cerr); err != nil {
		return nil, err
	}
	defer C.rm_item_ids_free(ids)
	n, err := goLength(count, "item count")
	if err != nil || n == 0 {
		return nil, err
	}
	view := unsafe.Slice(ids, n)
	out := make([]ItemID, n)
	for i, id := range view {
		out[i] = itemIDFromC(id)
	}
	return out, nil
}

// takeBytes copies bytes the C library handed over into Go memory and
// releases them. A size past math.MaxInt is refused as [ErrOutOfMemory].
func takeBytes(bytes *C.uint8_t, size C.size_t) ([]byte, error) {
	defer C.rm_bytes_free(bytes)
	n, err := goLength(size, "page size")
	if err != nil {
		return nil, err
	}
	out := make([]byte, n)
	if n > 0 {
		copy(out, unsafe.Slice((*byte)(unsafe.Pointer(bytes)), n))
	}
	return out, nil
}
