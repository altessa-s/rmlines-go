// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"slices"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

// The third-party pages of the corpus, and what each one holds: the one set
// of bytes here that no writer of ours produced.
func TestCorpusParsesToWhatItHolds(t *testing.T) {
	corpus := []struct {
		name       string
		strokes    int
		highlights int
		lines      []string
	}{
		{"Bold_Heading_Bullet_Normal.rm", 0, 0, []string{"A", "new line", "B is a letter of the alphabet", "C"}},
		{"Color_and_tool_v3.14.4.rm", 25, 6, nil},
		{"Lines_v2.rm", 10, 0, nil},
		{"Lines_v2_updated.rm", 10, 0, []string{""}},
		{"More_color_highlight_shader_v3.15.4.2.rm", 23, 0, nil},
		{"Normal_AB.rm", 0, 0, []string{"AB"}},
		{"Normal_A_stroke_2_layers.rm", 2, 0, []string{"A"}},
		{"Normal_A_stroke_2_layers_v3.2.2.rm", 3, 0, []string{"A", "v3.2.2"}},
		{"Normal_A_stroke_2_layers_v3.3.2.rm", 9, 0, []string{
			"A", "v3.2.2", "Normal bold italic", "Bold italic normal", "Bold line", "Normal line", "Heading line",
		}},
		{"Wikipedia_highlighted_p1.rm", 0, 4, nil},
		{"Wikipedia_highlighted_p2.rm", 0, 2, nil},
		{"With_SceneInfo_Block.rm", 13, 0, []string{"TEST TEXT", "Test text from keyboard"}},
		{"test-crdt-ordering.rm", 0, 0, []string{"A12_Z"}},
	}
	for _, want := range corpus {
		t.Run(want.name, func(t *testing.T) {
			page := mustParse(t, sampleBytes(t, want.name))
			if len(page.Strokes) != want.strokes {
				t.Errorf("strokes = %d, want %d", len(page.Strokes), want.strokes)
			}
			if len(page.Highlights) != want.highlights {
				t.Errorf("highlights = %d, want %d", len(page.Highlights), want.highlights)
			}
			if len(page.Images) != 0 {
				t.Errorf("images = %d, want 0", len(page.Images))
			}
			if page.SkippedBlocks != 0 {
				t.Errorf("skipped blocks = %d, want 0", page.SkippedBlocks)
			}
			if want.lines == nil {
				if page.Text != nil {
					t.Errorf("carries typed text, want none")
				}
				return
			}
			if page.Text == nil {
				t.Fatalf("carries no typed text")
			}
			var lines []string
			for _, p := range page.Text.Paragraphs {
				lines = append(lines, p.Text)
			}
			if !slices.Equal(lines, want.lines) {
				t.Errorf("lines = %q, want %q", lines, want.lines)
			}
		})
	}
}

func TestCorpusParagraphStylesMatchTheSampleName(t *testing.T) {
	page := mustParse(t, sampleBytes(t, "Bold_Heading_Bullet_Normal.rm"))
	want := []uint8{3, 2, 4, 1} // bold, heading, bullet, plain
	if page.Text == nil || len(page.Text.Paragraphs) != len(want) {
		t.Fatalf("want %d paragraphs", len(want))
	}
	for i, p := range page.Text.Paragraphs {
		if p.Format == nil || p.Format.Raw != want[i] {
			t.Errorf("paragraph %d format = %+v, want raw %d", i, p.Format, want[i])
		}
	}
}

func TestCorpusFaceRunsSurviveMovedText(t *testing.T) {
	page := mustParse(t, sampleBytes(t, "Normal_A_stroke_2_layers_v3.3.2.rm"))
	withFaces := 0
	for _, p := range page.Text.Paragraphs {
		if len(p.Faces) > 0 {
			withFaces++
		}
	}
	if withFaces != 2 {
		t.Errorf("lines with faces = %d, want 2", withFaces)
	}
}

func TestCorpusEveryMarkCarriesItsOwnIdentifier(t *testing.T) {
	for _, name := range samples {
		page := mustParse(t, sampleBytes(t, name))
		seen := map[rmlines.ItemID]bool{}
		for _, s := range page.Strokes {
			seen[s.ID] = true
		}
		for _, h := range page.Highlights {
			seen[h.ID] = true
		}
		if total := len(page.Strokes) + len(page.Highlights); len(seen) != total {
			t.Errorf("%s: %d distinct identifiers for %d marks", name, len(seen), total)
		}
	}
}
