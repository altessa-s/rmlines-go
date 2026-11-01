// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

func TestWrittenPageRoundTrips(t *testing.T) {
	page := roundTrip(t, sampleDraft())
	if page.Text == nil || len(page.Text.Paragraphs) != 3 {
		t.Fatalf("want three paragraphs, got %+v", page.Text)
	}
	formatted := 0
	for _, p := range page.Text.Paragraphs {
		if p.Format != nil {
			formatted++
		}
	}
	if formatted != 1 {
		t.Errorf("paragraphs with a style record = %d, want 1", formatted)
	}
	if got := page.Text.Paragraphs[0].Format.HeadingLevel(); got != 1 {
		t.Errorf("heading level = %d, want 1", got)
	}
	if faces := page.Text.Paragraphs[2].Faces; len(faces) != 1 || !faces[0].Bold || faces[0].Italic {
		t.Errorf("bold paragraph faces = %+v, want one bold run", faces)
	}
	if len(page.Strokes) != 2 || page.SkippedBlocks != 0 {
		t.Errorf("strokes = %d, skipped = %d; want 2 and 0", len(page.Strokes), page.SkippedBlocks)
	}
}

func TestLooseStrokeStaysLooseAndMarkStaysAnchored(t *testing.T) {
	page := roundTrip(t, sampleDraft())
	if len(page.Strokes) != 2 {
		t.Fatalf("strokes = %d, want 2", len(page.Strokes))
	}
	loose := page.Strokes[0]
	if loose.Anchor != nil || loose.Tool != rmlines.ToolFineliner2 || loose.Color != rmlines.ColorBlack ||
		len(loose.Points) != 3 {
		t.Errorf("loose stroke = %+v", loose)
	}
	if loose.Points[1].X != 50 || loose.Points[1].Y != 100 {
		t.Errorf("second point = %+v, want (50, 100)", loose.Points[1])
	}
	mark := page.Strokes[1]
	if mark.Anchor == nil || mark.Anchor.Paragraph != 1 {
		t.Fatalf("mark anchor = %+v, want paragraph 1", mark.Anchor)
	}
	if mark.Anchor.OriginX == nil || *mark.Anchor.OriginX != -464 {
		t.Errorf("mark origin = %v, want -464", mark.Anchor.OriginX)
	}
	if mark.Color != rmlines.ColorRed || len(mark.Points) != 2 {
		t.Errorf("mark = %+v", mark)
	}
	if page.AnchoredStrokes != 1 {
		t.Errorf("anchored strokes = %d, want 1", page.AnchoredStrokes)
	}
}

func TestImagesSurviveTheRoundTrip(t *testing.T) {
	d := sampleDraft()
	d.Images = []rmlines.DraftImage{{
		File: "0a418e1a-1111-4222-8333-444455556666.png", Key: imageKey(),
		X: -50, Y: 100, Width: 120, Height: 80,
	}}
	page := roundTrip(t, d)
	if len(page.Images) != 1 {
		t.Fatalf("images = %d, want 1", len(page.Images))
	}
	image := page.Images[0]
	if image.File != "0a418e1a-1111-4222-8333-444455556666.png" {
		t.Errorf("file = %q", image.File)
	}
	if image.Vertices[0].X != -50 || image.Vertices[0].Y != 100 {
		t.Errorf("first corner = %+v", image.Vertices[0])
	}
	if image.StrokesBefore != 2 || image.Anchor != nil {
		t.Errorf("strokes before = %d, anchor = %+v; want 2 and none", image.StrokesBefore, image.Anchor)
	}
}

func TestAnImageOnlyPageHasNoText(t *testing.T) {
	d := sampleDraft()
	d.Paragraphs, d.Strokes, d.Marks = nil, nil, nil
	d.Images = []rmlines.DraftImage{{File: "a.png", Key: imageKey(), X: -60, Y: 200, Width: 120, Height: 160}}
	page := roundTrip(t, d)
	if page.Text != nil || len(page.Images) != 1 || len(page.Strokes) != 0 {
		t.Errorf("text = %+v, images = %d, strokes = %d", page.Text, len(page.Images), len(page.Strokes))
	}
}

func TestAnAnchoredImageComesBackFastened(t *testing.T) {
	d := sampleDraft()
	d.Images = []rmlines.DraftImage{{
		File: "a.png", Key: imageKey(), X: 40, Y: 30, Width: 120, Height: 80,
		Anchor: &rmlines.DraftAnchor{Kind: rmlines.AnchorFirstCharacter, Paragraph: 1}, PenOrigin: -464,
	}}
	page := roundTrip(t, d)
	if len(page.Images) != 1 {
		t.Fatalf("images = %d, want 1", len(page.Images))
	}
	image := page.Images[0]
	if image.Anchor == nil || image.Anchor.Paragraph != 1 || image.Anchor.OriginX == nil ||
		*image.Anchor.OriginX != -464 {
		t.Errorf("anchor = %+v", image.Anchor)
	}
	if image.Vertices[0].X != 40 || image.Vertices[0].Y != 30 {
		t.Errorf("first corner = %+v, want the pen's (40, 30)", image.Vertices[0])
	}
}

func TestSeveralImagesKeepTheirOwnNames(t *testing.T) {
	d := sampleDraft()
	names := []string{"first.png", "second.png", "third.png", "fourth.png"}
	for i, name := range names {
		key := imageKey()
		key[15] = byte(0xE0 + i)
		d.Images = append(d.Images, rmlines.DraftImage{
			File: name, Key: key, X: float32(-50 + 10*i), Y: 100, Width: 120, Height: 80,
		})
	}
	page := roundTrip(t, d)
	if len(page.Images) != len(names) {
		t.Fatalf("images = %d, want %d", len(page.Images), len(names))
	}
	for i, image := range page.Images {
		if image.File != names[i] {
			t.Errorf("image %d = %q, want %q", i, image.File, names[i])
		}
	}
}

// Outside ASCII a length in bytes and a count of scalars are different
// numbers.
func TestTextOutsideASCIIRoundTrips(t *testing.T) {
	d := sampleDraft()
	d.Paragraphs = []rmlines.DraftParagraph{
		{Text: "Grüße, Welt", Style: ptr[uint8](2)},
		{Text: "Ελληνικά and English"},
		{Text: "絵文字 🖋 emoji"},
	}
	d.Marks[0].Anchor.Paragraph = 2
	d.Marks[0].Stroke.Points = d.Marks[0].Stroke.Points[:1]
	page := roundTrip(t, d)
	want := []string{"Grüße, Welt", "Ελληνικά and English", "絵文字 🖋 emoji"}
	if page.Text == nil || len(page.Text.Paragraphs) != len(want) {
		t.Fatalf("text = %+v", page.Text)
	}
	for i, p := range page.Text.Paragraphs {
		if p.Text != want[i] {
			t.Errorf("paragraph %d = %q, want %q", i, p.Text, want[i])
		}
	}
	if page.SkippedBlocks != 0 || len(page.Strokes) != 2 || page.Strokes[1].Anchor == nil ||
		page.Strokes[1].Anchor.Paragraph != 2 {
		t.Errorf("the mark did not find its paragraph past the non-ASCII lines")
	}
}

// Ill-formed bytes are written as one U+FFFD per maximal subpart, and counted
// as written.
func TestIllFormedTextIsWrittenAsReplacementCharacters(t *testing.T) {
	d := sampleDraft()
	d.Paragraphs = []rmlines.DraftParagraph{{Text: "a\x80b"}, {Text: "cd"}}
	d.Strokes = nil
	d.Marks = []rmlines.DraftMark{{
		Anchor: rmlines.DraftAnchor{Kind: rmlines.AnchorFirstCharacter, Paragraph: 1},
		Stroke: rmlines.DefaultStroke(rmlines.ToolFineliner2, rmlines.ColorBlack,
			[]rmlines.DraftPoint{rmlines.DefaultPoint(0, 0)}),
	}}
	page := roundTrip(t, d)
	if page.Text == nil || len(page.Text.Paragraphs) != 2 {
		t.Fatalf("text = %+v", page.Text)
	}
	if got := page.Text.Paragraphs[0].Text; got != "a�b" {
		t.Errorf("paragraph 0 = %q, want %q", got, "a�b")
	}
	if len(page.Strokes) != 1 || page.Strokes[0].Anchor == nil ||
		page.Strokes[0].Anchor.Paragraph != 1 || page.Strokes[0].Anchor.Offset != 1 {
		t.Errorf("mark anchor = %+v, want 1/1", page.Strokes[0].Anchor)
	}
}

func TestTheSentinelsResolve(t *testing.T) {
	stroke := rmlines.DefaultStroke(rmlines.ToolFineliner2, rmlines.ColorBlack,
		[]rmlines.DraftPoint{rmlines.DefaultPoint(0, 0)})
	d := sampleDraft()
	d.Paragraphs = []rmlines.DraftParagraph{{Text: "one"}, {Text: "two"}, {Text: "three"}, {Text: "four"}}
	d.Strokes = nil
	d.Marks = []rmlines.DraftMark{
		{Anchor: rmlines.DraftAnchor{Kind: rmlines.AnchorBeforeText}, Stroke: stroke},
		{Anchor: rmlines.DraftAnchor{Kind: rmlines.AnchorAfterText}, Stroke: stroke},
	}
	page := roundTrip(t, d)
	if len(page.Strokes) != 2 {
		t.Fatalf("strokes = %d, want 2", len(page.Strokes))
	}
	if a := page.Strokes[0].Anchor; a == nil || a.Paragraph != 0 || a.Offset != 0 {
		t.Errorf("before the text = %+v, want 0/0", a)
	}
	if a := page.Strokes[1].Anchor; a == nil || a.Paragraph != 3 || a.Offset != 5 {
		t.Errorf("after the text = %+v, want 3/5", a)
	}

	// With no text at all the sentinels still take ink.
	d.Paragraphs = nil
	if page := roundTrip(t, d); len(page.Strokes) != 2 {
		t.Errorf("strokes on a page without text = %d, want 2", len(page.Strokes))
	}
}

func TestDefaultsAreTheLibrarys(t *testing.T) {
	point := rmlines.DefaultPoint(1, 2)
	if point.X != 1 || point.Y != 2 || point.Width != 12 || point.Pressure != 6 {
		t.Errorf("default point = %+v, want width 12 and pressure 6", point)
	}
	points := []rmlines.DraftPoint{rmlines.DefaultPoint(0, 0), rmlines.DefaultPoint(10, 0)}
	stroke := rmlines.DefaultStroke(rmlines.ToolPencil2, rmlines.ColorBlue, points)
	if stroke.Tool != rmlines.ToolPencil2 || stroke.Color != rmlines.ColorBlue || stroke.ARGB != nil ||
		stroke.Thickness != 2 || len(stroke.Points) != 2 {
		t.Errorf("default stroke = %+v", stroke)
	}
	d := sampleDraft()
	d.Strokes = []rmlines.DraftStroke{stroke}
	d.Marks = nil
	page := roundTrip(t, d)
	if len(page.Strokes) != 1 || page.Strokes[0].ThicknessScale != 2 {
		t.Errorf("stroke = %+v, want thickness 2", page.Strokes)
	}
}

func TestAnAnchorPastTheTextIsRefused(t *testing.T) {
	d := sampleDraft()
	d.Marks[0].Anchor.Paragraph = 99
	data, err := rmlines.Write(d)
	var rmErr *rmlines.Error
	if !errors.As(err, &rmErr) || !errors.Is(err, rmlines.ErrAnchorOutOfRange) || rmErr.Value != 99 {
		t.Errorf("err = %v, want anchor out of range at 99", err)
	}
	if data != nil {
		t.Errorf("bytes were written")
	}

	d = sampleDraft()
	d.Images = []rmlines.DraftImage{{
		File: "a.png", Key: imageKey(), Width: 10, Height: 10,
		Anchor: &rmlines.DraftAnchor{Kind: rmlines.AnchorFirstCharacter, Paragraph: 42},
	}}
	_, err = rmlines.Write(d)
	if !errors.As(err, &rmErr) || rmErr.Status != rmlines.StatusAnchorOutOfRange || rmErr.Value != 42 {
		t.Errorf("err = %v, want anchor out of range at 42", err)
	}
}

func TestCountersThatWouldCollideAreRefused(t *testing.T) {
	stroke := rmlines.DefaultStroke(rmlines.ToolFineliner2, rmlines.ColorBlack,
		[]rmlines.DraftPoint{rmlines.DefaultPoint(0, 0)})
	const markLimit = 12500
	marks := make([]rmlines.DraftMark, markLimit+1)
	for i := range marks {
		marks[i] = rmlines.DraftMark{Anchor: rmlines.DraftAnchor{Kind: rmlines.AnchorBeforeText}, Stroke: stroke}
	}
	d := sampleDraft()
	d.Marks = marks
	_, err := rmlines.Write(d)
	var rmErr *rmlines.Error
	if !errors.As(err, &rmErr) || !errors.Is(err, rmlines.ErrTooManyToNumber) ||
		rmErr.Value != markLimit || rmErr.Message != "marks" {
		t.Errorf("err = %#v, want too many marks", err)
	}

	const characterLimit = 99980
	d = sampleDraft()
	d.Paragraphs = []rmlines.DraftParagraph{{Text: strings.Repeat("a", characterLimit)}}
	d.Marks = nil
	_, err = rmlines.Write(d)
	if !errors.As(err, &rmErr) || rmErr.Status != rmlines.StatusTooManyToNumber || rmErr.Value != characterLimit {
		t.Errorf("err = %#v, want too many characters", err)
	}
}

func TestTheWriterIsDeterministic(t *testing.T) {
	write := func(d *rmlines.Draft) []byte {
		t.Helper()
		data, err := rmlines.Write(d)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	faced := sampleDraft()
	faced.Paragraphs = []rmlines.DraftParagraph{{Text: "Bold", Bold: true}, {Text: "Italic", Italic: true}}
	faced.Marks = nil
	if !bytes.Equal(write(faced), write(faced)) {
		t.Errorf("a faced draft writes two different pages")
	}
	different := *faced
	different.Author = other
	if bytes.Equal(write(faced), write(&different)) {
		t.Errorf("a different author writes the same page")
	}
	withNamed := *faced
	withNamed.MarksAuthor = &named
	if !bytes.Equal(write(&withNamed), write(&withNamed)) || bytes.Equal(write(&withNamed), write(faced)) {
		t.Errorf("an explicit marks author is not used, or not deterministically")
	}
}

func TestAnEmptyDraftWritesAPage(t *testing.T) {
	data, err := rmlines.Write(&rmlines.Draft{
		Sheet:  rmlines.Sheet{ScreenWidth: 1404, ScreenHeight: 1872, CanvasWidth: 1404, CanvasHeight: 1872},
		Column: rmlines.ColumnOf(rmlines.ColumnMedium),
	})
	if err != nil || len(data) <= len(rmlines.Header) {
		t.Fatalf("write = %d bytes, %v", len(data), err)
	}
	mustParse(t, data)
}

func TestANulByteIsRefused(t *testing.T) {
	d := sampleDraft()
	d.Paragraphs[1].Text = "half\x00a paragraph"
	if _, err := rmlines.Write(d); !errors.Is(err, rmlines.ErrInvalidArgument) {
		t.Errorf("err = %v, want invalid argument", err)
	}
	d = sampleDraft()
	d.Images = []rmlines.DraftImage{{File: "a\x00.png", Width: 1, Height: 1}}
	if _, err := rmlines.Write(d); !errors.Is(err, rmlines.ErrInvalidArgument) {
		t.Errorf("err = %v, want invalid argument", err)
	}
	if _, err := rmlines.Write(nil); !errors.Is(err, rmlines.ErrInvalidArgument) {
		t.Errorf("err = %v, want invalid argument", err)
	}
}

func TestATemplateCanBeStampedOnAParsedPage(t *testing.T) {
	page := roundTrip(t, sampleDraft())
	if page.TemplateName != nil {
		t.Fatalf("a parsed page names a template")
	}
	stamped := page.WithTemplate("Grid")
	if stamped.TemplateName == nil || *stamped.TemplateName != "Grid" {
		t.Errorf("template = %v, want Grid", stamped.TemplateName)
	}
	if page.TemplateName != nil {
		t.Errorf("stamping changed the original")
	}
	if len(stamped.Strokes) != len(page.Strokes) || stamped.Text != page.Text {
		t.Errorf("the copy lost the page's content")
	}
	if stamped.WithoutTemplate().TemplateName != nil {
		t.Errorf("the template was not cleared")
	}
}
