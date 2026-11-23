// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

/*
#include "rmlines/rmlines.h"
void rmgo_session_log_buffer(uintptr_t handle);
*/
import "C"

import (
	"runtime"
	"runtime/cgo"
	"strings"
	"sync"
	"unsafe"
)

// Session owns an editable page and its undo history. Call Close to release
// both. Its methods are safe for concurrent use. Copies share the same history
// and lifetime. The zero value is closed.
type Session struct{ state *sessionState }
type sessionState struct {
	mu  sync.Mutex
	ptr *C.rm_edit_session
}

// EditLayer identifies a live root layer and its position-bearing scene link.
type EditLayer struct {
	Node, Link ItemID
	Visible    bool
}

// Transform maps (x,y) to (A*x+C*y+TX, B*x+D*y+TY) in node coordinates.
type Transform struct{ A, B, C, D, TX, TY float64 }

func (m Transform) toC() C.rm_edit_transform {
	return C.rm_edit_transform{a: C.double(m.A), b: C.double(m.B), c: C.double(m.C), d: C.double(m.D), tx: C.double(m.TX), ty: C.double(m.TY)}
}

// OpenSession opens a private copy of a complete page for editing.
func OpenSession(data []byte) (*Session, error) {
	var ptr *C.rm_edit_session
	var e C.rm_error
	if err := errorOf(C.rm_edit_open(bytesPointer(data), C.size_t(len(data)), &ptr, &e), &e); err != nil {
		return nil, err
	}
	return &Session{state: &sessionState{ptr: ptr}}, nil
}

// Close releases the session. Repeated calls are safe. After Close, history
// navigation returns false and operations returning errors report ErrInvalidArgument.
func (s *Session) Close() {
	if s == nil || s.state == nil {
		return
	}
	st := s.state
	st.mu.Lock()
	defer st.mu.Unlock()
	C.rm_edit_close(st.ptr)
	st.ptr = nil
}

func (s *Session) with(fn func(*C.rm_edit_session) error) error {
	if s == nil || s.state == nil {
		return ErrInvalidArgument
	}
	st := s.state
	st.mu.Lock()
	if st.ptr == nil {
		st.mu.Unlock()
		return ErrInvalidArgument
	}
	// The C sink runs on the calling thread. Capture its diagnostics for
	// this operation so handlers can use the session after it unlocks.
	var logs []func()
	handle := cgo.NewHandle(&logs)
	runtime.LockOSThread()
	C.rmgo_session_log_buffer(C.uintptr_t(handle))
	defer func() {
		C.rmgo_session_log_buffer(0)
		runtime.UnlockOSThread()
		handle.Delete()
		st.mu.Unlock()
		for _, deliver := range logs {
			runCallback(deliver)
		}
	}()
	return fn(st.ptr)
}

// Bytes returns a Go-owned snapshot of the current revision.
func (s *Session) Bytes() ([]byte, error) {
	var out []byte
	err := s.with(func(p *C.rm_edit_session) error {
		var size C.size_t
		b := C.rm_edit_bytes(p, &size)
		n, err := goLength(size, "page size")
		if err != nil {
			return err
		}
		out = append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(b)), n)...)
		return nil
	})
	return out, err
}

// Layers returns a snapshot of the live layers in drawing order.
func (s *Session) Layers() ([]EditLayer, error) {
	var out []EditLayer
	err := s.with(func(p *C.rm_edit_session) error {
		var a *C.rm_edit_layer
		var size C.size_t
		var e C.rm_error
		if err := errorOf(C.rm_edit_layers(p, &a, &size, &e), &e); err != nil {
			return err
		}
		defer C.rm_edit_layers_free(a)
		n, err := goLength(size, "layer count")
		if err != nil {
			return err
		}
		out = make([]EditLayer, n)
		for i, l := range unsafe.Slice(a, n) {
			out[i] = EditLayer{Node: itemIDFromC(l.node), Link: itemIDFromC(l.link), Visible: bool(l.visible)}
		}
		return nil
	})
	return out, err
}

func sessionIDs(ids []ItemID) []C.rm_item_id {
	out := make([]C.rm_item_id, len(ids))
	for i, id := range ids {
		out[i] = id.toC()
	}
	return out
}

// SetNodeName updates the node label with a fresh register stamp.
func (s *Session) SetNodeName(node, stamp ItemID, name string) error {
	return s.with(func(p *C.rm_edit_session) error {
		if strings.IndexByte(name, 0) >= 0 {
			return ErrInvalidArgument
		}
		var mem cMemory
		defer mem.free()
		var e C.rm_error
		return errorOf(C.rm_edit_set_node_name(p, node.toC(), stamp.toC(), mem.cString(name), &e), &e)
	})
}

// SetNodeVisible updates the node visibility with a fresh register stamp.
func (s *Session) SetNodeVisible(node, stamp ItemID, visible bool) error {
	return s.with(func(p *C.rm_edit_session) error {
		var e C.rm_error
		return errorOf(C.rm_edit_set_node_visible(p, node.toC(), stamp.toC(), C.bool(visible), &e), &e)
	})
}

// InsertStroke inserts a stroke between the supplied neighbours.
func (s *Session) InsertStroke(parent, id, left, right ItemID, stroke DraftStroke) error {
	return s.with(func(p *C.rm_edit_session) error {
		var mem cMemory
		defer mem.free()
		cs := mem.stroke(&stroke)
		var e C.rm_error
		return errorOf(C.rm_edit_insert_stroke(p, parent.toC(), id.toC(), left.toC(), right.toC(), &cs, &e), &e)
	})
}

// TransformStroke transforms one stroke using one fresh ID.
func (s *Session) TransformStroke(id, fresh ItemID, m Transform) error {
	return s.with(func(p *C.rm_edit_session) error {
		var e C.rm_error
		return errorOf(C.rm_edit_transform_stroke(p, id.toC(), fresh.toC(), m.toC(), &e), &e)
	})
}

// TransformStrokes transforms a selection atomically using consecutive IDs.
func (s *Session) TransformStrokes(ids []ItemID, first ItemID, m Transform) error {
	return s.with(func(p *C.rm_edit_session) error {
		items := sessionIDs(ids)
		var e C.rm_error
		return errorOf(C.rm_edit_transform_strokes(p, unsafe.SliceData(items), C.size_t(len(items)), first.toC(), m.toC(), &e), &e)
	})
}

// CutStroke removes a sample range and assigns consecutive IDs to surviving pieces.
func (s *Session) CutStroke(id, fresh ItemID, first, count int) error {
	return s.with(func(p *C.rm_edit_session) error {
		if first < 0 || count < 0 {
			return ErrInvalidArgument
		}
		var e C.rm_error
		return errorOf(C.rm_edit_cut_stroke(p, id.toC(), fresh.toC(), C.size_t(first), C.size_t(count), &e), &e)
	})
}

// Erase erases the selected items in one revision.
func (s *Session) Erase(ids []ItemID) error {
	return s.with(func(p *C.rm_edit_session) error {
		items := sessionIDs(ids)
		var e C.rm_error
		return errorOf(C.rm_edit_erase(p, unsafe.SliceData(items), C.size_t(len(items)), &e), &e)
	})
}

// ReplaceText replaces a Unicode scalar range, reserving one ID per inserted scalar
// plus one anchor stamp when the edit both removes and inserts scalars.
func (s *Session) ReplaceText(start, remove int, text string, first ItemID) error {
	return s.with(func(p *C.rm_edit_session) error {
		if start < 0 || remove < 0 || strings.IndexByte(text, 0) >= 0 {
			return ErrInvalidArgument
		}
		var mem cMemory
		defer mem.free()
		var e C.rm_error
		return errorOf(C.rm_edit_replace_text(p, C.size_t(start), C.size_t(remove), mem.cString(text), first.toC(), &e), &e)
	})
}

// AddLayer adds a layer at position using four consecutive IDs.
func (s *Session) AddLayer(position int, name string, first ItemID) error {
	return s.with(func(p *C.rm_edit_session) error {
		if position < 0 || strings.IndexByte(name, 0) >= 0 {
			return ErrInvalidArgument
		}
		var mem cMemory
		defer mem.free()
		var e C.rm_error
		return errorOf(C.rm_edit_add_layer(p, C.size_t(position), mem.cString(name), first.toC(), &e), &e)
	})
}

// MoveLayer moves a layer to position using one fresh link ID.
func (s *Session) MoveLayer(node ItemID, position int, first ItemID) error {
	return s.with(func(p *C.rm_edit_session) error {
		if position < 0 {
			return ErrInvalidArgument
		}
		var e C.rm_error
		return errorOf(C.rm_edit_move_layer(p, node.toC(), C.size_t(position), first.toC(), &e), &e)
	})
}

// DeleteLayer deletes a layer, reserving two consecutive IDs.
func (s *Session) DeleteLayer(node, first ItemID) error {
	return s.with(func(p *C.rm_edit_session) error {
		var e C.rm_error
		return errorOf(C.rm_edit_delete_layer(p, node.toC(), first.toC(), &e), &e)
	})
}

// Undo moves to the previous revision if one exists.
func (s *Session) Undo() bool {
	var moved bool
	_ = s.with(func(p *C.rm_edit_session) error { moved = bool(C.rm_edit_undo(p)); return nil })
	return moved
}

// Redo moves to the next revision if one exists.
func (s *Session) Redo() bool {
	var moved bool
	_ = s.with(func(p *C.rm_edit_session) error { moved = bool(C.rm_edit_redo(p)); return nil })
	return moved
}
