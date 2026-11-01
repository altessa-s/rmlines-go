// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

var (
	author = [16]byte{0x0a, 0x41, 0x8e, 0x1a, 0x11, 0x11, 0x42, 0x22, 0x83, 0x33, 0x44, 0x44, 0x55, 0x55, 0x66, 0x66}
	other  = [16]byte{0x0a, 0x41, 0x8e, 0x1a, 0x11, 0x11, 0x42, 0x22, 0x83, 0x33, 0x44, 0x44, 0x55, 0x55, 0x66, 0x77}
	named  = [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0x4d, 0xef, 0x80, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde}
)

// samples are the third-party pages of the corpus.
var samples = []string{
	"Bold_Heading_Bullet_Normal.rm",
	"Color_and_tool_v3.14.4.rm",
	"Lines_v2.rm",
	"Lines_v2_updated.rm",
	"More_color_highlight_shader_v3.15.4.2.rm",
	"Normal_AB.rm",
	"Normal_A_stroke_2_layers.rm",
	"Normal_A_stroke_2_layers_v3.2.2.rm",
	"Normal_A_stroke_2_layers_v3.3.2.rm",
	"Wikipedia_highlighted_p1.rm",
	"Wikipedia_highlighted_p2.rm",
	"With_SceneInfo_Block.rm",
	"test-crdt-ordering.rm",
}

func sampleBytes(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "v6-samples", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fixtureBytes(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustParse(t testing.TB, data []byte) *rmlines.Page {
	t.Helper()
	page, err := rmlines.Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return page
}

func ptr[T any](v T) *T { return &v }

// sampleDraft is a page with a heading, a plain and a bold paragraph, one
// loose stroke and one mark on the second paragraph.
func sampleDraft() *rmlines.Draft {
	return &rmlines.Draft{
		Author: author,
		Sheet:  rmlines.Sheet{ScreenWidth: 1404, ScreenHeight: 1872, CanvasWidth: 1404, CanvasHeight: 1872},
		Column: rmlines.Column{Width: 936, Setting: 1},
		Paragraphs: []rmlines.DraftParagraph{
			{Text: "Heading", Style: ptr[uint8](2)},
			{Text: "Plain body"},
			{Text: "Bold words", Bold: true},
		},
		Strokes: []rmlines.DraftStroke{{
			Tool: rmlines.ToolFineliner2, Color: rmlines.ColorBlack, Thickness: 2,
			Points: []rmlines.DraftPoint{
				rmlines.DefaultPoint(0, 100), rmlines.DefaultPoint(50, 100), rmlines.DefaultPoint(100, 100),
			},
		}},
		Marks: []rmlines.DraftMark{{
			Anchor:    rmlines.DraftAnchor{Kind: rmlines.AnchorFirstCharacter, Paragraph: 1},
			PenOrigin: -464,
			Stroke: rmlines.DraftStroke{
				Tool: rmlines.ToolFineliner2, Color: rmlines.ColorRed, Thickness: 2,
				Points: []rmlines.DraftPoint{rmlines.DefaultPoint(0, -10), rmlines.DefaultPoint(40, -10)},
			},
		}},
	}
}

func imageKey() [16]byte {
	var key [16]byte
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// roundTrip writes a draft and reads it straight back.
func roundTrip(t testing.TB, d *rmlines.Draft) *rmlines.Page {
	t.Helper()
	data, err := rmlines.Write(d)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	return mustParse(t, data)
}

// pageWithBlock is a page of the supported version holding one top-level
// block of the given type and payload.
func pageWithBlock(blockType byte, payload []byte) []byte {
	out := []byte(rmlines.Header)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(payload)))
	out = append(out, 0, 1, 1, blockType)
	return append(out, payload...)
}
