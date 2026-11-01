// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

func TestParseRefusals(t *testing.T) {
	padded := func(s string) []byte { return append([]byte(s), make([]byte, 64)...) }
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"nothing", nil, rmlines.ErrUnexpectedEOF},
		{"short", []byte("not a page"), rmlines.ErrUnexpectedEOF},
		{"a PDF", padded("%PDF-1.4 and then a good deal more than a header's worth"), rmlines.ErrUnsupportedFormat},
		{"version 5", padded("reMarkable .lines file, version=5          "), rmlines.ErrUnsupportedVersion},
	}
	for _, c := range cases {
		page, err := rmlines.Parse(c.data)
		if !errors.Is(err, c.want) || page != nil {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}

	_, err := rmlines.Parse(padded("reMarkable .lines file, version=3          "))
	var rmErr *rmlines.Error
	if !errors.As(err, &rmErr) || rmErr.Version != 3 || !strings.Contains(err.Error(), "version 3") {
		t.Errorf("err = %#v, want version 3 named", err)
	}
	if errors.Is(err, rmlines.ErrUnsupportedFormat) {
		t.Errorf("an older version reads as an unknown format")
	}

	_, err = rmlines.Parse(padded("%PDF-1.4 and then a good deal more than a header's worth"))
	if !errors.As(err, &rmErr) || !strings.HasPrefix(rmErr.Message, "%PDF-1.4") {
		t.Errorf("err = %#v, want the header's first bytes", err)
	}
}

// A block that will not read costs the block, not the page.
func TestALostStrokeIsCountedNotFatal(t *testing.T) {
	page := mustParse(t, pageWithBlock(0x05, make([]byte, 24)))
	if page.SkippedBlocks != 1 || len(page.Strokes) != 0 {
		t.Errorf("skipped = %d, strokes = %d; want 1 and 0", page.SkippedBlocks, len(page.Strokes))
	}
}

func TestSamplePagesKeepTheHighlightersOwnColours(t *testing.T) {
	page := mustParse(t, fixtureBytes(t, "highlighter-blue-orange.rm"))
	var blue, orange, withARGB, shader, shaderOK int
	for _, s := range page.Strokes {
		switch s.Tool {
		case rmlines.ToolHighlighter2:
			if s.Color == rmlines.ColorHighlighterBlue {
				blue++
			}
			if s.Color == rmlines.ColorHighlighterOrange {
				orange++
			}
			if s.ARGB != nil {
				withARGB++
			}
		case rmlines.ToolShader:
			shader++
			if s.Color == rmlines.ColorHighlight && s.ARGB != nil {
				shaderOK++
			}
		}
	}
	if blue != 2 || orange != 2 || withARGB != 0 || shader != 3 || shaderOK != 3 || page.SkippedBlocks != 0 {
		t.Errorf("blue %d, orange %d, argb %d, shader %d (%d with index 9 and argb)",
			blue, orange, withARGB, shader, shaderOK)
	}
}

func TestSamplePagesHighlightsInTheSecondFormKeepTheirColour(t *testing.T) {
	page := mustParse(t, fixtureBytes(t, "glyph-first-last-text.rm"))
	if len(page.Highlights) != 2 || page.SkippedBlocks != 0 {
		t.Fatalf("highlights = %d, skipped = %d", len(page.Highlights), page.SkippedBlocks)
	}
	for i, h := range page.Highlights {
		if h.Color != rmlines.ColorGreen || h.ARGB == nil || *h.ARGB != 0xFF000000 {
			t.Errorf("highlight %d: %v, %v", i, h.Color, h.ARGB)
		}
	}
	if page.Text == nil || page.Text.ColumnWidth == nil || *page.Text.ColumnWidth != 740 {
		t.Errorf("the column is not 740 wide")
	}
	if len(page.Text.Paragraphs) < 3 || page.Text.Paragraphs[2].Format == nil ||
		page.Text.Paragraphs[2].Format.Style() != rmlines.StyleCheckbox {
		t.Errorf("line 2 is not a checkbox")
	}
}

func TestSamplePageCheckboxesAreShownAsTheTabletShowsThem(t *testing.T) {
	page := mustParse(t, fixtureBytes(t, "checkbox-states.rm"))
	if page.Text == nil || len(page.Text.Paragraphs) < 10 {
		t.Fatalf("want ten lines at least")
	}
	text := page.Text
	lines := text.Paragraphs
	if f := lines[1].Format; f == nil || f.Style() != rmlines.StyleCheckbox || !f.HasTickedNestedItems() {
		t.Errorf("line 1: %+v, want style 6 with something nested ticked", f)
	}
	if f := lines[2].Format; f == nil || f.Style() != rmlines.StyleCheckboxCheckedNested {
		t.Errorf("line 2: %+v, want style 9", f)
	}
	if f := lines[6].Format; f == nil || f.Third != nil || f.HasTickedNestedItems() {
		t.Errorf("line 6: %+v, want no state", f)
	}
	want := []rmlines.CheckboxState{
		rmlines.CheckboxNone, rmlines.CheckboxTicked, rmlines.CheckboxTicked, rmlines.CheckboxPartial,
		rmlines.CheckboxUnticked, rmlines.CheckboxTicked, rmlines.CheckboxUnticked, rmlines.CheckboxUnticked,
		rmlines.CheckboxUnticked, rmlines.CheckboxUnticked,
	}
	for line, state := range want {
		if got := text.CheckboxState(line); got != state {
			t.Errorf("line %d: %v, want %v", line, got, state)
		}
	}
	for line := 1; line <= 5; line++ {
		level := 2
		if line == 1 || line == 3 {
			level = 1
		}
		if got := lines[line].Format.Level(); got != level {
			t.Errorf("line %d at level %d, want %d", line, got, level)
		}
	}
}

func TestThePiecesOfAnErasedShaderStrokeNameTheirSource(t *testing.T) {
	page := mustParse(t, fixtureBytes(t, "shader-erased.rm"))
	source := rmlines.ItemID{Author: 1, Counter: 17}
	var pieces, shaderPieces, without int
	for _, s := range page.Strokes {
		switch {
		case s.SourceID == nil:
			without++
		case *s.SourceID == source:
			pieces++
			if s.Tool == rmlines.ToolShader {
				shaderPieces++
			}
		}
		if s.ID == (rmlines.ItemID{Author: 1, Counter: 16}) && s.SourceID != nil {
			t.Errorf("the separate shader stroke names a source")
		}
	}
	if pieces != 4 || shaderPieces != 4 || without != len(page.Strokes)-4 {
		t.Errorf("pieces %d (%d shader), without %d of %d", pieces, shaderPieces, without, len(page.Strokes))
	}
}

func TestABallpointStrokeKeepsThePensSpeed(t *testing.T) {
	page := mustParse(t, fixtureBytes(t, "shader-erased.rm"))
	var points, moving, whole int
	for _, s := range page.Strokes {
		if s.Tool != rmlines.ToolBallpoint2 {
			continue
		}
		for _, p := range s.Points {
			points++
			if p.Speed > 0 {
				moving++
			}
			if p.Speed >= 0 && p.Speed <= 65535 && float32(int32(p.Speed)) == p.Speed {
				whole++
			}
		}
	}
	if points == 0 || moving == 0 || whole != points {
		t.Errorf("points %d, moving %d, whole %d", points, moving, whole)
	}
}

// Parse keeps no state between calls, so pages parse side by side.
func TestParseConcurrently(t *testing.T) {
	var inputs [][]byte
	var want []*rmlines.Page
	for _, name := range samples {
		data := sampleBytes(t, name)
		inputs = append(inputs, data)
		want = append(want, mustParse(t, data))
	}
	written, err := rmlines.Write(sampleDraft())
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for worker := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := range 8 {
				i := (worker + round) % len(inputs)
				page, err := rmlines.Parse(inputs[i])
				if err != nil {
					errs <- err
					return
				}
				if len(page.Strokes) != len(want[i].Strokes) || len(page.Highlights) != len(want[i].Highlights) {
					errs <- errors.New(samples[i] + " parsed differently under load")
					return
				}
				again, err := rmlines.Write(sampleDraft())
				if err != nil || !bytes.Equal(again, written) {
					errs <- errors.New("a concurrent write differs")
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
