// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

/*
#include "rmlines/rmlines.h"
*/
import "C"

import "unsafe"

// Rect is a rectangle in the page's own coordinates.
type Rect struct {
	X, Y, Width, Height float64
}

// Size is a sheet's width and height in the page's own units.
type Size struct {
	Width, Height float64
}

// StrokePoint is one sample of a stroke.
type StrokePoint struct {
	X, Y, Width, Pressure float32
	// Speed is the pen's speed as the file's uint16 stores it; the second
	// ballpoint thins its ink by it.
	Speed float32
}

// TextAnchor is the place in the typed text a stroke or an image is fastened
// to: a paragraph and a character, not a shift. Turning it into coordinates
// takes a font and a column width, and both belong to drawing.
type TextAnchor struct {
	// Paragraph is the index into [TextBlock.Paragraphs].
	Paragraph int32
	// Offset is how many of the paragraph's characters stand before the
	// anchor, counted in Unicode scalars, the way the file numbers them.
	Offset int32
	// OriginX is the horizontal the file itself declares, when it does.
	OriginX *float64
}

// Stroke is a single pen stroke.
type Stroke struct {
	// Tool is ToolUnknown when the file named a tool this package does not
	// know.
	Tool           PenTool
	Color          PenColor
	ThicknessScale float64
	Points         []StrokePoint
	// ARGB is the explicit colour, 0xAARRGGBB, when the page carries one.
	// Where it is set it wins over Color.
	ARGB *uint32
	// Anchor is the place in the typed text the stroke's coordinates are
	// counted from, when they are not counted from the sheet.
	Anchor *TextAnchor
	// ID is what the page calls the stroke: the handle an edit finds it by.
	ID ItemID
	// SourceID names the stroke this one is a piece of, when the eraser cut
	// a shader stroke. The tablet draws neighbouring pieces of one source as
	// one stroke.
	SourceID *ItemID
}

// Highlight is the marker over the text of an imported PDF: ready-made
// rectangles, not a stroke.
type Highlight struct {
	Rectangles []Rect
	Color      PenColor
	// ARGB is the explicit colour, when the page carries one; it wins over
	// Color.
	ARGB *uint32
	// Text is the words under the marker, as the tablet copied them out of
	// the text layer, when it did.
	Text *string
	// ID is what the page calls the highlight: the handle an edit finds it
	// by.
	ID ItemID
}

// ImageVertex is one corner of an image's quad: where it stands on the page
// and which point of the texture, 0 to 1, it pins there.
type ImageVertex struct {
	X, Y, U, V float64
}

// Image is a picture placed on the page. The pixels are not in the .rm: the
// file names an attachment that lies beside the page in the document.
type Image struct {
	// File is the attachment's file name, e.g. "<uuid>.png".
	File string
	// Vertices are the four corners in the order the file gives them; the
	// indexes 0 1 2 2 3 0 make a quad of them.
	Vertices [4]ImageVertex
	// StrokesBefore is how many strokes of the page are drawn before the
	// image: the z-order.
	StrokesBefore int
	// Anchor is where in the typed text the image is fastened, when it is;
	// its vertices are then counted from the pen at that anchor.
	Anchor *TextAnchor
}

// FaceRun is a stretch of a line set in bold or italic, in Unicode scalars of
// the line, half-open: [Start, End).
type FaceRun struct {
	Start, End   int32
	Bold, Italic bool
}

// Paragraph is one paragraph of typed text with everything the file says
// about it.
type Paragraph struct {
	Text string
	// Format is the paragraph's style record, when it has one.
	Format *ParagraphFormat
	// Faces are the bold and italic stretches of the line; nil for plain
	// text.
	Faces []FaceRun
}

// TextBlock is the typed text of a page. An image in the flow of the text is
// one U+FFFC of it, so that the offsets after it are the tablet's.
type TextBlock struct {
	// OriginX and OriginY are where the text begins: x counted from the
	// middle of the sheet, y growing downwards.
	OriginX, OriginY float64
	Paragraphs       []Paragraph
	// ColumnWidth is the declared width of the column, when the file
	// declares it.
	ColumnWidth *float64
}

// Page is one parsed page.
type Page struct {
	Strokes []Stroke
	// Highlights are the marker over the text of an imported PDF.
	Highlights []Highlight
	// Images are the pictures placed on the page, in the scene's order.
	Images []Image
	// Text is the typed text, when the page has any.
	Text *TextBlock
	// AnchoredStrokes is how many strokes are positioned from the text
	// rather than from the sheet.
	AnchoredStrokes int
	// SkippedBlocks counts the blocks that were not understood: unknown
	// types, and known ones whose body would not read. The service blocks
	// every page opens with are not a loss and are not counted.
	SkippedBlocks int
	// TemplateName is the page's template. A .rm knows nothing about its
	// background, so Parse never sets it; see [Page.WithTemplate].
	TemplateName *string
	// PaperSize is the sheet the page declares, when it declares one.
	PaperSize *Size
	// BackgroundVisible is whether the template is shown.
	BackgroundVisible bool
	// RootDocumentVisible is whether the imported document behind the page
	// is shown.
	RootDocumentVisible bool
}

// WithTemplate answers a copy of the page that names its template. The name
// comes from the document rather than from the page, so the layer that holds
// documents stamps it. The copy shares its slices with p.
func (p *Page) WithTemplate(name string) *Page {
	out := *p
	out.TemplateName = &name
	return &out
}

// WithoutTemplate answers a copy of the page that names no template. The copy
// shares its slices with p.
func (p *Page) WithoutTemplate() *Page {
	out := *p
	out.TemplateName = nil
	return &out
}

// Parse reads one page.
//
// A failure is an [*Error] with StatusUnsupportedVersion, StatusUnsupportedFormat,
// StatusUnexpectedEOF or StatusOutOfMemory. A block that will not read does
// not fail the page: it is counted in [Page.SkippedBlocks].
func Parse(data []byte) (*Page, error) {
	var page *C.rm_page
	var cerr C.rm_error
	status := C.rm_parse_page(bytesPointer(data), C.size_t(len(data)), &page, &cerr)
	if err := errorOf(status, &cerr); err != nil {
		return nil, err
	}
	defer C.rm_page_free(page)
	return pageFromC(page), nil
}

// emptyInput stands in for the bytes of an empty slice: the C library takes
// a null pointer for a caller's mistake, and an empty page is not one.
var emptyInput [1]byte

func bytesPointer(data []byte) *C.uint8_t {
	if len(data) == 0 {
		return (*C.uint8_t)(unsafe.Pointer(&emptyInput[0]))
	}
	return (*C.uint8_t)(unsafe.Pointer(&data[0]))
}

func goString(s C.rm_str) string {
	if s.length == 0 {
		return ""
	}
	return C.GoStringN(s.bytes, C.int(s.length))
}

func itemIDFromC(id C.rm_item_id) ItemID {
	return ItemID{Author: uint8(id.author), Counter: uint64(id.counter)}
}

func anchorFromC(a *C.rm_text_anchor) *TextAnchor {
	out := &TextAnchor{Paragraph: int32(a.paragraph), Offset: int32(a.offset)}
	if a.has_origin_x {
		x := float64(a.origin_x)
		out.OriginX = &x
	}
	return out
}

func optionalARGB(has C.bool, argb C.uint32_t) *uint32 {
	if !has {
		return nil
	}
	v := uint32(argb)
	return &v
}

func pageFromC(p *C.rm_page) *Page {
	out := &Page{
		AnchoredStrokes:     int(p.anchored_strokes),
		SkippedBlocks:       int(p.skipped_blocks),
		BackgroundVisible:   bool(p.background_visible),
		RootDocumentVisible: bool(p.root_document_visible),
	}
	if p.has_template_name {
		name := goString(p.template_name)
		out.TemplateName = &name
	}
	if p.has_paper_size {
		out.PaperSize = &Size{Width: float64(p.paper_size.width), Height: float64(p.paper_size.height)}
	}
	if p.stroke_count > 0 {
		strokes := unsafe.Slice(p.strokes, p.stroke_count)
		out.Strokes = make([]Stroke, len(strokes))
		for i := range strokes {
			out.Strokes[i] = strokeFromC(&strokes[i])
		}
	}
	if p.highlight_count > 0 {
		highlights := unsafe.Slice(p.highlights, p.highlight_count)
		out.Highlights = make([]Highlight, len(highlights))
		for i := range highlights {
			out.Highlights[i] = highlightFromC(&highlights[i])
		}
	}
	if p.image_count > 0 {
		images := unsafe.Slice(p.images, p.image_count)
		out.Images = make([]Image, len(images))
		for i := range images {
			out.Images[i] = imageFromC(&images[i])
		}
	}
	if p.has_text {
		out.Text = textFromC(&p.text)
	}
	return out
}

func strokeFromC(s *C.rm_stroke) Stroke {
	out := Stroke{
		Tool:           PenTool(s.tool),
		Color:          PenColor(s.color),
		ThicknessScale: float64(s.thickness_scale),
		ARGB:           optionalARGB(s.has_argb, s.argb),
		ID:             itemIDFromC(s.id),
	}
	if s.point_count > 0 {
		points := unsafe.Slice(s.points, s.point_count)
		out.Points = make([]StrokePoint, len(points))
		for i, p := range points {
			out.Points[i] = StrokePoint{
				X: float32(p.x), Y: float32(p.y), Width: float32(p.width),
				Pressure: float32(p.pressure), Speed: float32(p.speed),
			}
		}
	}
	if s.has_anchor {
		out.Anchor = anchorFromC(&s.anchor)
	}
	if s.has_source_id {
		source := itemIDFromC(s.source_id)
		out.SourceID = &source
	}
	return out
}

func highlightFromC(h *C.rm_highlight) Highlight {
	out := Highlight{
		Color: PenColor(h.color),
		ARGB:  optionalARGB(h.has_argb, h.argb),
		ID:    itemIDFromC(h.id),
	}
	if h.rectangle_count > 0 {
		rects := unsafe.Slice(h.rectangles, h.rectangle_count)
		out.Rectangles = make([]Rect, len(rects))
		for i, r := range rects {
			out.Rectangles[i] = Rect{
				X: float64(r.x), Y: float64(r.y), Width: float64(r.width), Height: float64(r.height),
			}
		}
	}
	if h.has_text {
		text := goString(h.text)
		out.Text = &text
	}
	return out
}

func imageFromC(m *C.rm_page_image) Image {
	out := Image{File: goString(m.file), StrokesBefore: int(m.strokes_before)}
	for i, v := range m.vertices {
		out.Vertices[i] = ImageVertex{X: float64(v.x), Y: float64(v.y), U: float64(v.u), V: float64(v.v)}
	}
	if m.has_anchor {
		out.Anchor = anchorFromC(&m.anchor)
	}
	return out
}

func textFromC(t *C.rm_text_block) *TextBlock {
	out := &TextBlock{OriginX: float64(t.origin_x), OriginY: float64(t.origin_y)}
	if t.has_column_width {
		width := float64(t.column_width)
		out.ColumnWidth = &width
	}
	if t.paragraph_count > 0 {
		paragraphs := unsafe.Slice(t.paragraphs, t.paragraph_count)
		out.Paragraphs = make([]Paragraph, len(paragraphs))
		for i := range paragraphs {
			p := &paragraphs[i]
			para := Paragraph{Text: goString(p.text)}
			if p.has_format {
				format := formatFromC(&p.format)
				para.Format = &format
			}
			if p.face_count > 0 {
				faces := unsafe.Slice(p.faces, p.face_count)
				para.Faces = make([]FaceRun, len(faces))
				for j, f := range faces {
					para.Faces[j] = FaceRun{
						Start: int32(f.start), End: int32(f.end), Bold: bool(f.bold), Italic: bool(f.italic),
					}
				}
			}
			out.Paragraphs[i] = para
		}
	}
	return out
}

// HeaderVersion answers the version a header declares, and false when the
// bytes are not a header at all. The version is read rather than compared
// against known spellings, so a generation nobody has written down yet is
// named by its number.
func HeaderVersion(header []byte) (int, bool) {
	var version C.int
	ok := C.rm_header_version((*C.char)(unsafe.Pointer(bytesPointer(header))), C.size_t(len(header)), &version)
	if !ok {
		return 0, false
	}
	return int(version), true
}
