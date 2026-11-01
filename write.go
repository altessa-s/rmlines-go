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
	"strings"
	"unsafe"
)

// Sheet is the sheet a page declares, the ways the tablet declares it. Which
// sheet it is decides where absolute units land.
type Sheet struct {
	// ScreenWidth and ScreenHeight are the size as whole units.
	ScreenWidth, ScreenHeight uint32
	// CanvasWidth and CanvasHeight are the same size unrounded, and the one
	// the format calls authoritative.
	CanvasWidth, CanvasHeight float64
}

// Column is the typed-text column of a draft: its width in the file's own
// units and which of the tablet's three settings that is. The two have to
// agree; [ColumnOf] is how a caller avoids inventing a fourth.
type Column struct {
	// Width is the column's width in the file's own units.
	Width float32
	// Setting is which of the three settings it is: 0, 1 or 2.
	Setting uint8
}

// ColumnOf answers the draft column of one of the three settings.
func ColumnOf(column TextColumn) Column {
	c := C.rm_draft_column_of(C.rm_text_column(column))
	return Column{Width: float32(c.width), Setting: uint8(c.setting)}
}

// DraftParagraph is one paragraph of a draft's typed text.
type DraftParagraph struct {
	// Text is UTF-8 and must not contain a newline: the writer puts the
	// newlines in. Ill-formed bytes are written as U+FFFD.
	Text string
	// Style is the tag 1 value; nil writes no style record, which is plain
	// body text.
	Style *uint8
	// Second and Third are tag 2 and tag 3 of the style record.
	Second *uint8
	Third  *uint32
	// Bold and Italic set the whole paragraph in that face.
	Bold, Italic bool
}

// DraftPoint is one sample of a stroke, with the raw numbers the file stores
// beside the coordinates. The drawn half-width is Width / 4 in sheet units;
// the writer stores, it does not interpret.
type DraftPoint struct {
	// X and Y are the position in sheet units, or from the pen for a mark.
	X, Y float32
	// Speed, Width, Direction and Pressure are stored as given.
	Speed     uint16
	Width     uint16
	Direction uint8
	Pressure  uint8
}

// DefaultPoint answers a point at x, y with the defaults a point usually
// carries: a width of 12 and a pressure of 6.
func DefaultPoint(x, y float32) DraftPoint {
	return pointFromDraft(C.rm_draft_point_make(C.float(x), C.float(y)))
}

func pointFromDraft(p C.rm_draft_point) DraftPoint {
	return DraftPoint{
		X: float32(p.x), Y: float32(p.y), Speed: uint16(p.speed), Width: uint16(p.width),
		Direction: uint8(p.direction), Pressure: uint8(p.pressure),
	}
}

// DraftStroke is one stroke of a draft: the tool, the colour and the points as
// written.
type DraftStroke struct {
	// Tool is the tool the stroke is drawn with.
	Tool PenTool
	// Color is the palette index.
	Color PenColor
	// ARGB is the explicit colour, 0xAARRGGBB, set only where an index
	// cannot name it: the shader's colours.
	ARGB *uint32
	// Thickness is the setting on the tablet, 1, 2 or 3, and not a factor:
	// whatever it does to the line is already in each point's width.
	Thickness float64
	// Points are written in order; nil writes a stroke without points.
	Points []DraftPoint
}

// DefaultStroke answers a stroke with the defaults a stroke usually carries:
// no ARGB and a thickness of 2. The points are not copied.
func DefaultStroke(tool PenTool, color PenColor, points []DraftPoint) DraftStroke {
	made := C.rm_draft_stroke_make(C.rm_pen_tool(tool), C.rm_pen_color(color), nil, 0)
	return DraftStroke{
		Tool:      PenTool(made.tool),
		Color:     PenColor(made.color),
		ARGB:      optionalARGB(made.has_argb, made.argb),
		Thickness: float64(made.thickness),
		Points:    points,
	}
}

// AnchorKind is what an anchored stroke or image hangs on.
type AnchorKind int32

// The anchor kinds.
const (
	// AnchorFirstCharacter is the first character of a paragraph.
	AnchorFirstCharacter AnchorKind = 0
	// AnchorBeforeText is the sentinel before all of the text.
	AnchorBeforeText AnchorKind = 1
	// AnchorAfterText is the sentinel just past the last character typed, in
	// the paragraph the last Return closes.
	AnchorAfterText AnchorKind = 2
)

// DraftAnchor is where a mark or an image hangs on the typed text.
type DraftAnchor struct {
	// Kind is what the anchor hangs on.
	Kind AnchorKind
	// Paragraph is read only for AnchorFirstCharacter.
	Paragraph int32
}

// DraftMark is a stroke hung on the typed text rather than on the sheet: its
// coordinates are counted from the pen, so the ink follows the text when it
// reflows.
type DraftMark struct {
	// Anchor is where the stroke hangs.
	Anchor DraftAnchor
	// PenOrigin is where the pen stands horizontally. The end of a line is the
	// layout's to know and the format has no layout, so the number comes from
	// the caller.
	PenOrigin float32
	// Stroke is the ink, its points counted from the pen.
	Stroke DraftStroke
}

// DraftImage is a picture placed on the page. The pixels are not in the .rm;
// the page carries the attachment's name, its 16-byte key and the placement.
type DraftImage struct {
	// File is the attachment's file name, extension included.
	File string
	// Key is the 16 opaque bytes the item and the attachment map share.
	Key [16]byte
	// X, Y, Width and Height are the axis-aligned placement in sheet units,
	// or counted from the pen at the anchor for an anchored image.
	X, Y, Width, Height float32
	// Anchor fastens the image to the typed text rather than to the sheet,
	// when set.
	Anchor *DraftAnchor
	// PenOrigin is where the pen stands, read only when Anchor is set.
	PenOrigin float32
}

// Draft is a page to be written, in the same vocabulary [Parse] answers in.
type Draft struct {
	// Author is the page's author, as a document's .content spells its UUID.
	Author [16]byte
	// MarksAuthor is the author the bold and italic marks are written under.
	// Nil derives one from Author, the same way every time, so that one
	// draft always writes the same bytes.
	MarksAuthor *[16]byte
	// Sheet is the sheet the page declares.
	Sheet Sheet
	// Column is the typed-text column; see [ColumnOf].
	Column Column
	// Paragraphs are the typed text; none writes a page without text.
	Paragraphs []DraftParagraph
	// Strokes hang on nothing, in the sheet's own coordinates.
	Strokes []DraftStroke
	// Marks hang on the typed text.
	Marks []DraftMark
	// Images stand at the end of the drawing order, over the strokes.
	Images []DraftImage
}

// Write answers the whole .rm file of one fresh page. Nothing is written on
// failure.
//
// A mark or an image fastened to a paragraph the draft does not have is
// refused with an error matching [ErrAnchorOutOfRange]; more items of one kind
// than a page can number, with [ErrTooManyToNumber]. A paragraph text or an
// image file name holding a NUL byte is refused with [ErrInvalidArgument].
func Write(d *Draft) ([]byte, error) {
	if d == nil {
		return nil, &Error{Status: StatusInvalidArgument, Message: "draft"}
	}
	var mem cMemory
	defer mem.free()

	draft := (*C.rm_page_draft)(mem.alloc(1, unsafe.Sizeof(C.rm_page_draft{})))
	for i, b := range d.Author {
		draft.author[i] = C.uint8_t(b)
	}
	if d.MarksAuthor != nil {
		draft.has_marks_author = true
		for i, b := range d.MarksAuthor {
			draft.marks_author[i] = C.uint8_t(b)
		}
	}
	draft.sheet = C.rm_draft_sheet{
		screen_width:  C.uint32_t(d.Sheet.ScreenWidth),
		screen_height: C.uint32_t(d.Sheet.ScreenHeight),
		canvas_width:  C.double(d.Sheet.CanvasWidth),
		canvas_height: C.double(d.Sheet.CanvasHeight),
	}
	draft.column = C.rm_draft_column{width: C.float(d.Column.Width), setting: C.uint8_t(d.Column.Setting)}

	if n := len(d.Paragraphs); n > 0 {
		paragraphs := unsafe.Slice((*C.rm_draft_paragraph)(mem.alloc(n, unsafe.Sizeof(C.rm_draft_paragraph{}))), n)
		for i, p := range d.Paragraphs {
			if strings.IndexByte(p.Text, 0) >= 0 {
				return nil, &Error{Status: StatusInvalidArgument, Message: "paragraphs.text"}
			}
			paragraphs[i].text = mem.cString(p.Text)
			if p.Style != nil {
				paragraphs[i].has_style = true
				paragraphs[i].style = C.uint8_t(*p.Style)
			}
			if p.Second != nil {
				paragraphs[i].has_second = true
				paragraphs[i].second = C.uint8_t(*p.Second)
			}
			if p.Third != nil {
				paragraphs[i].has_third = true
				paragraphs[i].third = C.uint32_t(*p.Third)
			}
			paragraphs[i].bold = C.bool(p.Bold)
			paragraphs[i].italic = C.bool(p.Italic)
		}
		draft.paragraphs = &paragraphs[0]
		draft.paragraph_count = C.size_t(n)
	}

	if n := len(d.Strokes); n > 0 {
		strokes := unsafe.Slice((*C.rm_draft_stroke)(mem.alloc(n, unsafe.Sizeof(C.rm_draft_stroke{}))), n)
		for i := range d.Strokes {
			strokes[i] = mem.stroke(&d.Strokes[i])
		}
		draft.strokes = &strokes[0]
		draft.stroke_count = C.size_t(n)
	}

	if n := len(d.Marks); n > 0 {
		marks := unsafe.Slice((*C.rm_draft_mark)(mem.alloc(n, unsafe.Sizeof(C.rm_draft_mark{}))), n)
		for i := range d.Marks {
			m := &d.Marks[i]
			marks[i].anchor = draftAnchorToC(m.Anchor)
			marks[i].pen_origin = C.float(m.PenOrigin)
			marks[i].stroke = mem.stroke(&m.Stroke)
		}
		draft.marks = &marks[0]
		draft.mark_count = C.size_t(n)
	}

	if n := len(d.Images); n > 0 {
		images := unsafe.Slice((*C.rm_draft_image)(mem.alloc(n, unsafe.Sizeof(C.rm_draft_image{}))), n)
		for i := range d.Images {
			m := &d.Images[i]
			if strings.IndexByte(m.File, 0) >= 0 {
				return nil, &Error{Status: StatusInvalidArgument, Message: "images.file"}
			}
			images[i].file = mem.cString(m.File)
			for k, b := range m.Key {
				images[i].key[k] = C.uint8_t(b)
			}
			images[i].x = C.float(m.X)
			images[i].y = C.float(m.Y)
			images[i].width = C.float(m.Width)
			images[i].height = C.float(m.Height)
			if m.Anchor != nil {
				images[i].has_anchor = true
				images[i].anchor = draftAnchorToC(*m.Anchor)
				images[i].pen_origin = C.float(m.PenOrigin)
			}
		}
		draft.images = &images[0]
		draft.image_count = C.size_t(n)
	}

	var out *C.uint8_t
	var size C.size_t
	var cerr C.rm_error
	status := C.rm_write_page(draft, &out, &size, &cerr)
	if err := errorOf(status, &cerr); err != nil {
		return nil, err
	}
	return takeBytes(out, size)
}

func draftAnchorToC(a DraftAnchor) C.rm_draft_anchor {
	return C.rm_draft_anchor{kind: C.rm_anchor_kind(a.Kind), paragraph: C.int32_t(a.Paragraph)}
}

// cMemory is the C memory one call builds its arguments in. The draft is a
// graph of pointers, and cgo forbids handing C Go memory that holds Go
// pointers, so the whole graph lives in C memory and is freed as one.
type cMemory struct {
	blocks []unsafe.Pointer
}

// alloc answers n zeroed elements of the given size. It panics when C is out
// of memory, as a Go allocation would.
func (m *cMemory) alloc(n int, size uintptr) unsafe.Pointer {
	p := C.calloc(C.size_t(n), C.size_t(size))
	if p == nil {
		panic("rmlines: out of C memory")
	}
	m.blocks = append(m.blocks, p)
	return p
}

func (m *cMemory) cString(s string) *C.char {
	p := C.CString(s)
	m.blocks = append(m.blocks, unsafe.Pointer(p))
	return p
}

func (m *cMemory) stroke(s *DraftStroke) C.rm_draft_stroke {
	out := C.rm_draft_stroke{
		tool:      C.rm_pen_tool(s.Tool),
		color:     C.rm_pen_color(s.Color),
		thickness: C.double(s.Thickness),
	}
	if s.ARGB != nil {
		out.has_argb = true
		out.argb = C.uint32_t(*s.ARGB)
	}
	if n := len(s.Points); n > 0 {
		points := unsafe.Slice((*C.rm_draft_point)(m.alloc(n, unsafe.Sizeof(C.rm_draft_point{}))), n)
		for i, p := range s.Points {
			points[i] = C.rm_draft_point{
				x: C.float(p.X), y: C.float(p.Y), speed: C.uint16_t(p.Speed), width: C.uint16_t(p.Width),
				direction: C.uint8_t(p.Direction), pressure: C.uint8_t(p.Pressure),
			}
		}
		out.points = &points[0]
		out.point_count = C.size_t(n)
	}
	return out
}

func (m *cMemory) free() {
	for _, p := range m.blocks {
		C.free(p)
	}
	m.blocks = nil
}
