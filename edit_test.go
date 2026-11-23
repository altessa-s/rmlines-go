// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"bytes"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestSessionEditingAndHistory(t *testing.T) {
	d := sampleDraft()
	data, err := rmlines.Write(d)
	if err != nil {
		t.Fatal(err)
	}
	s, err := rmlines.OpenSession(data)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := func(counter uint64) rmlines.ItemID { return rmlines.ItemID{Author: 1, Counter: counter} }
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	snap, err := s.Bytes()
	check(err)
	layers, err := s.Layers()
	check(err)
	if len(layers) != 1 {
		t.Fatal(layers)
	}
	node := id(300000)
	zero := rmlines.ItemID{}
	check(s.AddLayer(0, "Overlay", node))
	check(s.MoveLayer(node, 1, id(300005)))
	check(s.SetNodeName(node, id(300010), "Paint"))
	check(s.SetNodeVisible(node, id(300011), false))
	check(s.SetNodeVisible(node, id(300012), true))
	check(s.InsertStroke(node, id(300020), zero, zero, d.Strokes[0]))
	m := rmlines.Transform{A: 1, D: 1, TX: 3}
	check(s.TransformStroke(id(300020), id(300021), m))
	check(s.CutStroke(id(300021), id(300030), 1, 1))
	check(s.TransformStrokes([]rmlines.ItemID{id(300030), id(300031)}, id(300040), m))
	check(s.Erase([]rmlines.ItemID{id(300040), id(300041)}))
	check(s.ReplaceText(0, 1, "h", id(300050)))
	check(s.DeleteLayer(node, id(300060)))
	if !s.Undo() {
		t.Fatal("undo")
	}
	if err := s.Erase([]rmlines.ItemID{id(900000)}); !errors.Is(err, rmlines.ErrItemsNotFound) || !s.Redo() {
		t.Fatal("failed edit or redo", err)
	}
	if err := s.SetNodeName(node, id(400000), "bad\x00name"); !errors.Is(err, rmlines.ErrInvalidArgument) {
		t.Fatal(err)
	}
	if err := s.ReplaceText(0, 0, "bad\x00text", id(400000)); !errors.Is(err, rmlines.ErrInvalidArgument) {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			_, _ = s.Bytes()
		}
	}()
	s.Close()
	<-done
	s.Close()
	if _, err := s.Layers(); !errors.Is(err, rmlines.ErrInvalidArgument) || s.Undo() || s.Redo() {
		t.Fatal("closed session", err)
	}
	if !bytes.Equal(snap, data) {
		t.Fatal("snapshot changed")
	}
}

func TestSessionLoggerReadsSnapshot(t *testing.T) {
	s, err := rmlines.OpenSession(sampleBytes(t, "Normal_A_stroke_2_layers.rm"))
	if err != nil {
		t.Fatal(err)
	}
	layers, err := s.Layers()
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	rmlines.SetLogger(func(_ rmlines.LogLevel, _ string) {
		if _, err := s.Bytes(); err != nil {
			t.Error(err)
		}
		calls.Add(1)
	})
	defer rmlines.SetLogger(nil)
	done := make(chan error, 1)
	go func() {
		for i, l := range layers {
			if err := s.SetNodeVisible(l.Node, rmlines.ItemID{Author: 1, Counter: uint64(900000 + i)}, false); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case err := <-done:
		s.Close()
		if err != nil || calls.Load() == 0 {
			t.Fatalf("edit error %v, callbacks %d", err, calls.Load())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("session logger blocked during edit")
	}
}

func TestCopiedSessionSharesHistoryAndClose(t *testing.T) {
	s, err := rmlines.OpenSession(sampleBytes(t, "Normal_A_stroke_2_layers.rm"))
	if err != nil {
		t.Fatal(err)
	}
	other := *s
	layers, err := other.Layers()
	if err != nil {
		t.Fatal(err)
	}
	check := func(err error) {
		if err != nil && !errors.Is(err, rmlines.ErrInvalidArgument) {
			t.Error(err)
		}
	}
	var workers, started sync.WaitGroup
	start := make(chan struct{})
	for _, work := range []func(int){
		func(i int) {
			check(s.SetNodeVisible(layers[0].Node, rmlines.ItemID{Author: 1, Counter: uint64(900000 + i)}, false))
		},
		func(int) { other.Undo(); other.Redo() },
		func(int) {
			_, err := other.Bytes()
			check(err)
			_, err = other.Layers()
			check(err)
		},
	} {
		started.Add(1)
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			work(0)
			started.Done()
			for i := 1; i <= 50; i++ {
				work(i)
			}
		}()
	}
	close(start)
	started.Wait()
	other.Close()
	workers.Wait()
	s.Close()
	if _, err := s.Bytes(); !errors.Is(err, rmlines.ErrInvalidArgument) {
		t.Fatalf("original after close through copy: %v", err)
	}
}

func TestReplaceTextPreservesHandwritingAnchor(t *testing.T) {
	draft := rmlines.Draft{
		Sheet:      rmlines.Sheet{ScreenWidth: 1404, ScreenHeight: 1872, CanvasWidth: 1404, CanvasHeight: 1872},
		Column:     rmlines.Column{Width: 936, Setting: 1},
		Paragraphs: []rmlines.DraftParagraph{{Text: "Intro"}, {Text: "XYZ"}},
		Marks: []rmlines.DraftMark{{
			Anchor: rmlines.DraftAnchor{Kind: rmlines.AnchorFirstCharacter, Paragraph: 1},
			Stroke: rmlines.DraftStroke{Tool: rmlines.ToolFineliner2, Thickness: 2, Points: []rmlines.DraftPoint{rmlines.DefaultPoint(0, 0), rmlines.DefaultPoint(40, 0)}},
		}},
	}
	data, err := rmlines.Write(&draft)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range []string{"X", "Q"} {
		t.Run(replacement, func(t *testing.T) {
			s, err := rmlines.OpenSession(data)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.ReplaceText(6, 1, replacement, rmlines.ItemID{Author: 1, Counter: 900000}); err != nil {
				t.Fatal(err)
			}
			b, err := s.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			p := mustParse(t, b)
			if len(p.Text.Paragraphs) != 2 || p.Text.Paragraphs[1].Text != replacement+"YZ" || len(p.Strokes) != 1 {
				t.Fatalf("unexpected paragraphs or strokes after replacing X with %s", replacement)
			}
			if a := p.Strokes[0].Anchor; a == nil || a.Paragraph != 1 || a.Offset != 1 {
				t.Errorf("anchor = %+v, want paragraph 1 offset 1", a)
			}
		})
	}
}
