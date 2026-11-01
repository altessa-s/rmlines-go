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
	"unsafe"
)

// PenTool is the tool a stroke was made with. The numbers are the tablet's
// own; ToolUnknown stands for a number this package has not seen.
type PenTool int32

// The tools.
const (
	ToolUnknown           PenTool = -1
	ToolPaintbrush1       PenTool = 0
	ToolPencil1           PenTool = 1
	ToolBallpoint1        PenTool = 2
	ToolMarker1           PenTool = 3
	ToolFineliner1        PenTool = 4
	ToolHighlighter1      PenTool = 5
	ToolEraser            PenTool = 6
	ToolMechanicalPencil1 PenTool = 7
	ToolEraserArea        PenTool = 8
	ToolPaintbrush2       PenTool = 12
	ToolMechanicalPencil2 PenTool = 13
	ToolPencil2           PenTool = 14
	ToolBallpoint2        PenTool = 15
	ToolMarker2           PenTool = 16
	ToolFineliner2        PenTool = 17
	ToolHighlighter2      PenTool = 18
	ToolCalligraphy       PenTool = 21
	ToolShader            PenTool = 23
)

var penToolNames = map[PenTool]string{
	ToolUnknown: "unknown", ToolPaintbrush1: "paintbrush-1", ToolPencil1: "pencil-1",
	ToolBallpoint1: "ballpoint-1", ToolMarker1: "marker-1", ToolFineliner1: "fineliner-1",
	ToolHighlighter1: "highlighter-1", ToolEraser: "eraser",
	ToolMechanicalPencil1: "mechanical-pencil-1", ToolEraserArea: "eraser-area",
	ToolPaintbrush2: "paintbrush-2", ToolMechanicalPencil2: "mechanical-pencil-2",
	ToolPencil2: "pencil-2", ToolBallpoint2: "ballpoint-2", ToolMarker2: "marker-2",
	ToolFineliner2: "fineliner-2", ToolHighlighter2: "highlighter-2",
	ToolCalligraphy: "calligraphy", ToolShader: "shader",
}

func (t PenTool) String() string {
	if name, ok := penToolNames[t]; ok {
		return name
	}
	return "tool(" + strconv.Itoa(int(t)) + ")"
}

// IsEraser reports whether strokes of the tool must not be drawn: an eraser's
// strokes look like ordinary lines, exactly where the user was erasing.
func (t PenTool) IsEraser() bool {
	return bool(C.rm_pen_tool_is_eraser(C.rm_pen_tool(t)))
}

// IsHighlighter reports whether the tool is the marker. Over handwriting, and
// over a scan with no text layer, its band is a stroke and the only record
// that anything was marked.
func (t PenTool) IsHighlighter() bool {
	return bool(C.rm_pen_tool_is_highlighter(C.rm_pen_tool(t)))
}

// PenColor is an index into the tablet's palette. How an index is coloured
// belongs to a renderer.
type PenColor int32

// The palette indexes.
const (
	ColorBlack       PenColor = 0
	ColorGray        PenColor = 1
	ColorWhite       PenColor = 2
	ColorYellow      PenColor = 3
	ColorGreen       PenColor = 4
	ColorPink        PenColor = 5
	ColorBlue        PenColor = 6
	ColorRed         PenColor = 7
	ColorGrayOverlap PenColor = 8
	ColorHighlight   PenColor = 9
	ColorGreen2      PenColor = 10
	ColorCyan        PenColor = 11
	ColorMagenta     PenColor = 12
	ColorYellow2     PenColor = 13
	// The highlighter's blue and orange, under indexes of their own. Some
	// pages spell the same two colours as ColorHighlight plus an ARGB.
	ColorHighlighterBlue   PenColor = 14
	ColorHighlighterOrange PenColor = 15
)

var penColorNames = [...]string{
	"black", "gray", "white", "yellow", "green", "pink", "blue", "red", "gray-overlap",
	"highlight", "green-2", "cyan", "magenta", "yellow-2", "highlighter-blue",
	"highlighter-orange",
}

func (c PenColor) String() string {
	if c >= 0 && int(c) < len(penColorNames) {
		return penColorNames[c]
	}
	return "color(" + strconv.Itoa(int(c)) + ")"
}

// ParagraphStyle is how a paragraph of typed text is presented. The numbers
// are the tablet's own; StyleUnknown stands for a value this package does not
// know, which is read as plain text.
type ParagraphStyle int32

// The paragraph styles.
const (
	StyleUnknown               ParagraphStyle = -1
	StyleBasic                 ParagraphStyle = 0
	StylePlain                 ParagraphStyle = 1
	StyleHeading               ParagraphStyle = 2
	StyleSubheading            ParagraphStyle = 3
	StyleBullet                ParagraphStyle = 4
	StyleBulletNested          ParagraphStyle = 5
	StyleCheckbox              ParagraphStyle = 6
	StyleCheckboxChecked       ParagraphStyle = 7
	StyleCheckboxNested        ParagraphStyle = 8
	StyleCheckboxCheckedNested ParagraphStyle = 9
	StyleNumbered              ParagraphStyle = 10
	StyleNumberedNested        ParagraphStyle = 11
	// StyleImageBlock holds an image in the flow of the text: one U+FFFC
	// between two newlines.
	StyleImageBlock ParagraphStyle = 12
)

var paragraphStyleNames = [...]string{
	"basic", "plain", "heading", "subheading", "bullet", "bullet-nested", "checkbox",
	"checkbox-checked", "checkbox-nested", "checkbox-checked-nested", "numbered",
	"numbered-nested", "image-block",
}

func (s ParagraphStyle) String() string {
	if s >= 0 && int(s) < len(paragraphStyleNames) {
		return paragraphStyleNames[s]
	}
	return "unknown"
}

// ListFamily answers which family of list the style belongs to, whatever its
// level.
func (s ParagraphStyle) ListFamily() ListFamily {
	return ListFamily(C.rm_paragraph_style_list_family(C.rm_paragraph_style(s)))
}

// IsListItem reports whether the paragraph is an item of a list: those get a
// marker drawn in front of them.
func (s ParagraphStyle) IsListItem() bool {
	return bool(C.rm_paragraph_style_is_list_item(C.rm_paragraph_style(s)))
}

// ListFamily is the kind of list a paragraph style belongs to.
type ListFamily int32

// The list families.
const (
	ListNone     ListFamily = -1
	ListBullet   ListFamily = 0
	ListCheckbox ListFamily = 1
	ListNumbered ListFamily = 2
)

func (f ListFamily) String() string {
	switch f {
	case ListBullet:
		return "bullet"
	case ListCheckbox:
		return "checkbox"
	case ListNumbered:
		return "numbered"
	default:
		return "none"
	}
}

// CheckboxState is how the checkbox of a list item is shown. CheckboxNone
// stands for a line that is not an item of a checkbox list.
type CheckboxState int32

// The checkbox states.
const (
	CheckboxNone     CheckboxState = -1
	CheckboxUnticked CheckboxState = 0
	CheckboxTicked   CheckboxState = 1
	// CheckboxPartial: some of the nested items are ticked and some are not.
	CheckboxPartial CheckboxState = 2
)

func (s CheckboxState) String() string {
	switch s {
	case CheckboxUnticked:
		return "unticked"
	case CheckboxTicked:
		return "ticked"
	case CheckboxPartial:
		return "partial"
	default:
		return "none"
	}
}

// TextColumn is the width of the typed-text column: one of the three settings
// the tablet offers.
type TextColumn int32

// The column settings.
const (
	ColumnNarrow TextColumn = 0
	ColumnMedium TextColumn = 1
	ColumnWide   TextColumn = 2
)

func (c TextColumn) String() string {
	switch c {
	case ColumnNarrow:
		return "narrow"
	case ColumnMedium:
		return "medium"
	case ColumnWide:
		return "wide"
	default:
		return "column(" + strconv.Itoa(int(c)) + ")"
	}
}

// Width answers the width of the column in the file's own units.
func (c TextColumn) Width() float32 {
	return float32(C.rm_text_column_width(C.rm_text_column(c)))
}

// Setting answers the value the scene block stores for the column: the
// setting, not the width.
func (c TextColumn) Setting() uint8 {
	return uint8(C.rm_text_column_setting(C.rm_text_column(c)))
}

// WidestFittingColumn answers the widest column that stands on a sheet of
// this width with a margin. The margin is the whole of the rule: a column
// that merely fits runs into both edges.
func WidestFittingColumn(sheetWidth float64) TextColumn {
	return TextColumn(C.rm_text_column_widest_fitting(C.double(sheetWidth)))
}

// ParagraphFormat is the whole style record of a paragraph, not merely its
// first field: two paragraphs of one Raw value can be drawn differently, and
// what follows Raw is what tells them apart.
type ParagraphFormat struct {
	// Raw is tag 1 as it stands in the file, known to this package or not.
	Raw uint8
	// Second is tag 2, when the record carries one.
	Second *uint8
	// Third is tag 3, when the record carries one.
	Third *uint32
}

func (f ParagraphFormat) toC() C.rm_paragraph_format {
	out := C.rm_paragraph_format{raw: C.uint8_t(f.Raw)}
	if f.Second != nil {
		out.has_second = true
		out.second = C.uint8_t(*f.Second)
	}
	if f.Third != nil {
		out.has_third = true
		out.third = C.uint32_t(*f.Third)
	}
	return out
}

func formatFromC(f *C.rm_paragraph_format) ParagraphFormat {
	out := ParagraphFormat{Raw: uint8(f.raw)}
	if f.has_second {
		second := uint8(f.second)
		out.Second = &second
	}
	if f.has_third {
		third := uint32(f.third)
		out.Third = &third
	}
	return out
}

// Style answers the kind of paragraph, or StyleUnknown for a Raw value this
// package does not know.
func (f ParagraphFormat) Style() ParagraphStyle {
	c := f.toC()
	return ParagraphStyle(C.rm_paragraph_format_style(&c))
}

// HasFurtherFields reports whether the record carries anything past tag 1.
func (f ParagraphFormat) HasFurtherFields() bool {
	c := f.toC()
	return bool(C.rm_paragraph_format_has_further_fields(&c))
}

// HeadingLevel answers which of the three heading levels the paragraph is, or
// 0 when it is not a heading.
func (f ParagraphFormat) HeadingLevel() int {
	c := f.toC()
	return int(C.rm_paragraph_format_heading_level(&c))
}

// Level answers how deep a list item is nested, counting from one.
func (f ParagraphFormat) Level() int {
	c := f.toC()
	return int(C.rm_paragraph_format_level(&c))
}

// IsTicked reports whether the item's checkbox is ticked: by its style, or by
// the tick bit of tag 3 on an item with nested items.
func (f ParagraphFormat) IsTicked() bool {
	c := f.toC()
	return bool(C.rm_paragraph_format_is_ticked(&c))
}

// HasTickedNestedItems reports whether any item nested under this one is
// ticked, by the second state bit of tag 3.
func (f ParagraphFormat) HasTickedNestedItems() bool {
	c := f.toC()
	return bool(C.rm_paragraph_format_has_ticked_nested_items(&c))
}

// CheckboxState answers how the checkbox of a paragraph is shown, or
// CheckboxNone when the paragraph is not an item of a checkbox list or is past
// the last one. Without the state bits in the file, the nested items
// themselves are read.
func (t *TextBlock) CheckboxState(line int) CheckboxState {
	if t == nil || line < 0 || line >= len(t.Paragraphs) {
		return CheckboxNone
	}
	// Only the formats are read; the text stays behind.
	count := len(t.Paragraphs)
	paragraphs := (*C.rm_paragraph)(C.calloc(C.size_t(count), C.size_t(unsafe.Sizeof(C.rm_paragraph{}))))
	if paragraphs == nil {
		return CheckboxNone
	}
	defer C.free(unsafe.Pointer(paragraphs))
	view := unsafe.Slice(paragraphs, count)
	for i, p := range t.Paragraphs {
		if p.Format != nil {
			view[i].has_format = true
			view[i].format = p.Format.toC()
		}
	}
	block := C.rm_text_block{paragraphs: paragraphs, paragraph_count: C.size_t(count)}
	return CheckboxState(C.rm_text_block_checkbox_state(&block, C.size_t(line)))
}
