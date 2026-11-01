// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

func TestAHighlightIsErasedAndTheRestStays(t *testing.T) {
	data := sampleBytes(t, "Wikipedia_highlighted_p1.rm")
	page := mustParse(t, data)
	if len(page.Highlights) < 2 {
		t.Fatalf("the sample has %d highlights", len(page.Highlights))
	}
	target := page.Highlights[1].ID
	edited, err := rmlines.EraseItems(data, []rmlines.ItemID{target})
	if err != nil {
		t.Fatal(err)
	}
	after := mustParse(t, edited)
	if len(after.Highlights) != len(page.Highlights)-1 || after.SkippedBlocks != 0 {
		t.Fatalf("highlights = %d, skipped = %d", len(after.Highlights), after.SkippedBlocks)
	}
	var kept []rmlines.ItemID
	for _, h := range page.Highlights {
		if h.ID != target {
			kept = append(kept, h.ID)
		}
	}
	var got []rmlines.ItemID
	for _, h := range after.Highlights {
		got = append(got, h.ID)
	}
	if !slices.Equal(got, kept) {
		t.Errorf("highlights = %v, want %v", got, kept)
	}
	if len(edited) >= len(data) {
		t.Errorf("the page did not shrink: %d -> %d bytes", len(data), len(edited))
	}
}

func TestAStrokeIsErased(t *testing.T) {
	data := sampleBytes(t, "Color_and_tool_v3.14.4.rm")
	page := mustParse(t, data)
	edited, err := rmlines.EraseItems(data, []rmlines.ItemID{page.Strokes[2].ID})
	if err != nil {
		t.Fatal(err)
	}
	after := mustParse(t, edited)
	if len(after.Strokes) != len(page.Strokes)-1 || len(after.Highlights) != len(page.Highlights) {
		t.Errorf("strokes %d -> %d, highlights %d -> %d", len(page.Strokes), len(after.Strokes),
			len(page.Highlights), len(after.Highlights))
	}
}

func TestEverythingOnEverySampleErasedAtOnceReadsBackEmpty(t *testing.T) {
	for _, name := range samples {
		data := sampleBytes(t, name)
		page := mustParse(t, data)
		var ids []rmlines.ItemID
		for _, s := range page.Strokes {
			ids = append(ids, s.ID)
		}
		for _, h := range page.Highlights {
			ids = append(ids, h.ID)
		}
		edited, err := rmlines.EraseItems(data, ids)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		after := mustParse(t, edited)
		if len(after.Strokes) != 0 || len(after.Highlights) != 0 || after.SkippedBlocks != page.SkippedBlocks {
			t.Errorf("%s: %d strokes, %d highlights, %d skipped left", name,
				len(after.Strokes), len(after.Highlights), after.SkippedBlocks)
		}
		if (after.Text == nil) != (page.Text == nil) {
			t.Errorf("%s: the text did not stay", name)
		}
	}
}

func TestAMissingIdentifierRefusesTheWholeEdit(t *testing.T) {
	data := sampleBytes(t, "Wikipedia_highlighted_p1.rm")
	page := mustParse(t, data)
	missing := rmlines.ItemID{Author: 9, Counter: 999999}
	edited, err := rmlines.EraseItems(data, []rmlines.ItemID{page.Highlights[0].ID, missing})
	var rmErr *rmlines.Error
	if !errors.Is(err, rmlines.ErrItemsNotFound) || !errors.As(err, &rmErr) || rmErr.Value != 1 ||
		rmErr.Message != "not on the page: 9:999999" {
		t.Errorf("err = %#v, want one missing, named", err)
	}
	if edited != nil {
		t.Errorf("bytes were handed over")
	}
}

func TestAnErasedItemIsNotThereToEraseAgain(t *testing.T) {
	data := sampleBytes(t, "Wikipedia_highlighted_p1.rm")
	target := mustParse(t, data).Highlights[0].ID
	once, err := rmlines.EraseItems(data, []rmlines.ItemID{target})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rmlines.EraseItems(once, []rmlines.ItemID{target}); !errors.Is(err, rmlines.ErrItemsNotFound) {
		t.Errorf("err = %v, want items not found", err)
	}
}

func TestNothingAskedNothingChanged(t *testing.T) {
	data := sampleBytes(t, "Color_and_tool_v3.14.4.rm")
	edited, err := rmlines.EraseItems(data, nil)
	if err != nil || !bytes.Equal(edited, data) {
		t.Errorf("an empty edit changed the page (%v)", err)
	}
}

func TestEraseRefusesWhatTheParserRefuses(t *testing.T) {
	if _, err := rmlines.EraseItems([]byte("not a page"), []rmlines.ItemID{{Author: 1, Counter: 1}}); !errors.Is(err, rmlines.ErrUnexpectedEOF) {
		t.Errorf("err = %v, want unexpected EOF", err)
	}
}

func TestHighlightsOverATextLayerAreFound(t *testing.T) {
	data := sampleBytes(t, "Wikipedia_highlighted_p1.rm")
	page := mustParse(t, data)
	ids, err := rmlines.HighlighterItems(data)
	if err != nil {
		t.Fatal(err)
	}
	var want []rmlines.ItemID
	for _, h := range page.Highlights {
		want = append(want, h.ID)
	}
	if !slices.Equal(ids, want) {
		t.Errorf("highlighter items = %v, want %v", ids, want)
	}

	// Everything the marker did, erased, leaves no marker.
	edited, err := rmlines.EraseItems(data, ids)
	if err != nil {
		t.Fatal(err)
	}
	if left, err := rmlines.HighlighterItems(edited); err != nil || len(left) != 0 {
		t.Errorf("left = %v, %v; want none", left, err)
	}
}

func TestHighlighterStrokesAreFound(t *testing.T) {
	data := fixtureBytes(t, "highlighter-blue-orange.rm")
	page := mustParse(t, data)
	ids, err := rmlines.HighlighterItems(data)
	if err != nil {
		t.Fatal(err)
	}
	var want []rmlines.ItemID
	for _, s := range page.Strokes {
		if s.Tool.IsHighlighter() {
			want = append(want, s.ID)
		}
	}
	if len(want) == 0 || !slices.Equal(ids, want) {
		t.Errorf("highlighter items = %v, want %v", ids, want)
	}
}

func TestHighlighterItemsOfAPageWithoutMarker(t *testing.T) {
	ids, err := rmlines.HighlighterItems(sampleBytes(t, "Lines_v2.rm"))
	if err != nil || ids != nil {
		t.Errorf("ids = %v, err = %v; want none", ids, err)
	}
}

func TestHighlighterItemsRefuseAnIncompletePage(t *testing.T) {
	// A stroke block whose payload is not a stroke.
	_, err := rmlines.HighlighterItems(pageWithBlock(0x05, make([]byte, 24)))
	var rmErr *rmlines.Error
	if !errors.Is(err, rmlines.ErrIncompletePage) || !errors.As(err, &rmErr) || rmErr.Value != 1 {
		t.Errorf("err = %#v, want an incomplete page losing one block", err)
	}
}
