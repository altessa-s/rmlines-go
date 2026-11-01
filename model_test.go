// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"errors"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

func TestItemIDSpellsAsAuthorColonCounter(t *testing.T) {
	id := rmlines.ItemID{Author: 2, Counter: 1234}
	if got := id.String(); got != "2:1234" {
		t.Errorf("spelling = %q, want 2:1234", got)
	}
	if back, err := rmlines.ParseItemID("2:1234"); err != nil || !back.Equal(id) {
		t.Errorf("parse = %v, %v", back, err)
	}
	widest, err := rmlines.ParseItemID("255:18446744073709551615")
	if err != nil || widest.Author != 255 || widest.Counter != ^uint64(0) {
		t.Fatalf("widest = %v, %v", widest, err)
	}
	if s := widest.String(); s != "255:18446744073709551615" || len(s) != rmlines.MaxItemIDLength {
		t.Errorf("widest spells %q (%d characters)", s, len(s))
	}

	accepted := map[string]rmlines.ItemID{
		"+2:+1":  {Author: 2, Counter: 1},
		"-0:-00": {Author: 0, Counter: 0},
	}
	for text, want := range accepted {
		if got, err := rmlines.ParseItemID(text); err != nil || got != want {
			t.Errorf("%q = %v, %v; want %v", text, got, err, want)
		}
	}
	refused := []string{
		"2:", ":1", "256:1", "2:1:3", "-1:2", " 2:1", "2:+", "2:+-1", "2:-1",
		"2:18446744073709551616", "", "21", "2:1\x00",
	}
	for _, text := range refused {
		if _, err := rmlines.ParseItemID(text); !errors.Is(err, rmlines.ErrInvalidArgument) {
			t.Errorf("%q: err = %v, want invalid argument", text, err)
		}
	}
}

func TestTheVersionIsReadOutOfTheHeader(t *testing.T) {
	cases := map[string]int{
		"reMarkable .lines file, version=3          ": 3,
		"reMarkable .lines file, version=5          ": 5,
		rmlines.Header: 6,
		"reMarkable .lines file, version=7          ": 7,
	}
	for header, want := range cases {
		if got, ok := rmlines.HeaderVersion([]byte(header)); !ok || got != want {
			t.Errorf("%q = %d, %v; want %d", header, got, ok, want)
		}
	}
	for _, header := range []string{
		"reMarkable lines with selections and layers", "reMarkable .lines file, version=           ", "%PDF-1.4", "",
	} {
		if _, ok := rmlines.HeaderVersion([]byte(header)); ok {
			t.Errorf("%q read as a header", header)
		}
	}
}

func TestTheColumnLeavesAShareOfTheSheet(t *testing.T) {
	cases := map[float64]rmlines.TextColumn{
		954: rmlines.ColumnNarrow, 1404: rmlines.ColumnWide, 1620: rmlines.ColumnWide,
		1200: rmlines.ColumnMedium, 1: rmlines.ColumnNarrow, 0: rmlines.ColumnNarrow,
	}
	for width, want := range cases {
		if got := rmlines.WidestFittingColumn(width); got != want {
			t.Errorf("sheet %v: %v, want %v", width, got, want)
		}
	}
}

func TestWidthsAndSettingsAgree(t *testing.T) {
	cases := []struct {
		column  rmlines.TextColumn
		width   float32
		setting uint8
	}{
		{rmlines.ColumnNarrow, 740, 0}, {rmlines.ColumnMedium, 936, 1}, {rmlines.ColumnWide, 1152, 2},
	}
	for _, c := range cases {
		if c.column.Width() != c.width || c.column.Setting() != c.setting {
			t.Errorf("%v: %v/%v, want %v/%v", c.column, c.column.Width(), c.column.Setting(), c.width, c.setting)
		}
		if got := rmlines.ColumnOf(c.column); got != (rmlines.Column{Width: c.width, Setting: c.setting}) {
			t.Errorf("ColumnOf(%v) = %+v", c.column, got)
		}
	}
}

func format(raw uint8, second, third int64) rmlines.ParagraphFormat {
	f := rmlines.ParagraphFormat{Raw: raw}
	if second >= 0 {
		f.Second = ptr(uint8(second))
	}
	if third >= 0 {
		f.Third = ptr(uint32(third))
	}
	return f
}

func TestHeadingLevelsComeFromTheWholeRecord(t *testing.T) {
	cases := []struct {
		format rmlines.ParagraphFormat
		level  int
	}{
		{format(2, -1, -1), 1}, {format(3, 2, 3), 2}, {format(3, -1, -1), 3}, {format(1, -1, -1), 0},
	}
	for _, c := range cases {
		if got := c.format.HeadingLevel(); got != c.level {
			t.Errorf("%+v: heading level %d, want %d", c.format, got, c.level)
		}
	}
}

func TestListLevelsComeFromTheLowNibble(t *testing.T) {
	cases := []struct {
		format rmlines.ParagraphFormat
		level  int
	}{
		{format(4, -1, -1), 1}, {format(6, -1, -1), 1},
		{format(5, 3, 0x01), 2}, {format(5, 3, 0x02), 3},
		{format(8, 3, 0x11), 2}, {format(8, 3, 0x12), 3},
		{format(11, 3, 0x21), 2}, {format(11, 3, 0x22), 3},
		{format(1, 3, 0x12), 1},
	}
	for _, c := range cases {
		if got := c.format.Level(); got != c.level {
			t.Errorf("%+v: level %d, want %d", c.format, got, c.level)
		}
	}
}

func TestAnUnknownStyleSurvivesAsItsRawValue(t *testing.T) {
	unseen := format(99, 7, 7)
	if unseen.Style() != rmlines.StyleUnknown || !unseen.HasFurtherFields() || unseen.HeadingLevel() != 0 ||
		unseen.Level() != 1 {
		t.Errorf("unknown style read as %v", unseen.Style())
	}
}

func TestListFamiliesGroupTheStyles(t *testing.T) {
	families := map[rmlines.ParagraphStyle]rmlines.ListFamily{
		rmlines.StyleBullet: rmlines.ListBullet, rmlines.StyleBulletNested: rmlines.ListBullet,
		rmlines.StyleCheckbox: rmlines.ListCheckbox, rmlines.StyleCheckboxChecked: rmlines.ListCheckbox,
		rmlines.StyleCheckboxNested: rmlines.ListCheckbox, rmlines.StyleCheckboxCheckedNested: rmlines.ListCheckbox,
		rmlines.StyleNumbered: rmlines.ListNumbered, rmlines.StyleNumberedNested: rmlines.ListNumbered,
		rmlines.StylePlain: rmlines.ListNone, rmlines.StyleHeading: rmlines.ListNone,
		rmlines.StyleImageBlock: rmlines.ListNone,
	}
	for style, want := range families {
		if got := style.ListFamily(); got != want {
			t.Errorf("%v: family %v, want %v", style, got, want)
		}
		if style.IsListItem() != (want != rmlines.ListNone) {
			t.Errorf("%v: IsListItem = %v", style, style.IsListItem())
		}
	}
}

func TestTheToolsAnswerTheTwoQuestionsAboutThem(t *testing.T) {
	for tool, want := range map[rmlines.PenTool][2]bool{
		rmlines.ToolEraser: {true, false}, rmlines.ToolEraserArea: {true, false},
		rmlines.ToolFineliner2: {false, false}, rmlines.ToolHighlighter1: {false, true},
		rmlines.ToolHighlighter2: {false, true}, rmlines.ToolShader: {false, false},
		rmlines.ToolUnknown: {false, false},
	} {
		if tool.IsEraser() != want[0] || tool.IsHighlighter() != want[1] {
			t.Errorf("%v: eraser %v, highlighter %v; want %v", tool, tool.IsEraser(), tool.IsHighlighter(), want)
		}
	}
}

func TestCheckboxStatesWithoutStateBitsReadTheNestedItems(t *testing.T) {
	text := &rmlines.TextBlock{Paragraphs: []rmlines.Paragraph{
		{Text: "parent", Format: ptr(format(6, -1, -1))},
		{Text: "child", Format: ptr(format(9, 3, 0x011))},
		{Text: "old-style ticked", Format: ptr(format(7, -1, -1))},
	}}
	want := []rmlines.CheckboxState{rmlines.CheckboxPartial, rmlines.CheckboxTicked, rmlines.CheckboxTicked,
		rmlines.CheckboxNone}
	for line, state := range want {
		if got := text.CheckboxState(line); got != state {
			t.Errorf("line %d: %v, want %v", line, got, state)
		}
	}
	if got := text.CheckboxState(-1); got != rmlines.CheckboxNone {
		t.Errorf("line -1: %v", got)
	}
	nested := format(9, 3, 0x12)
	if nested.Style() != rmlines.StyleCheckboxCheckedNested || nested.Level() != 3 || !nested.IsTicked() {
		t.Errorf("a nested ticked checkbox reads as %v at level %d", nested.Style(), nested.Level())
	}
}

// The Go spelling of every constant is the C library's.
func TestConstantsAreTheLibrarys(t *testing.T) {
	version, header, supported, ink := rmlines.CConstants()
	if version != rmlines.Version || header != rmlines.Header || supported != rmlines.SupportedVersion {
		t.Errorf("C says %q %q %d", version, header, supported)
	}
	want := [4]float64{rmlines.InkUnitsPerRawWidth, rmlines.InkMinimumStrokeWidth, rmlines.InkMaximumStrokeWidth,
		rmlines.InkFaintAlpha}
	if ink != want {
		t.Errorf("ink scales %v, want %v", ink, want)
	}
}

func TestStatusesAndEnumsHaveNames(t *testing.T) {
	if rmlines.StatusItemsNotFound.String() != "items the page does not hold as living ones" {
		t.Errorf("status = %q", rmlines.StatusItemsNotFound.String())
	}
	for _, s := range []interface{ String() string }{
		rmlines.ToolShader, rmlines.ColorHighlighterOrange, rmlines.StyleImageBlock, rmlines.ListNumbered,
		rmlines.CheckboxPartial, rmlines.ColumnWide, rmlines.LogError,
	} {
		if s.String() == "" {
			t.Errorf("%#v has no name", s)
		}
	}
}

func TestErrorIsMatchesByStatus(t *testing.T) {
	err := error(&rmlines.Error{Status: rmlines.StatusItemsNotFound, Value: 2, Message: "not on the page: 1:2"})
	if !errors.Is(err, rmlines.ErrItemsNotFound) {
		t.Errorf("an error does not match the sentinel of its status")
	}
	if errors.Is(err, rmlines.ErrUnverified) {
		t.Errorf("an error matches the sentinel of another status")
	}
	if !errors.Is(rmlines.ErrUnverified, rmlines.ErrUnverified) {
		t.Errorf("a sentinel does not match itself")
	}
	var typedNil *rmlines.Error
	if errors.Is(err, typedNil) {
		t.Errorf("an error matches a typed nil")
	}
	if errors.Is(typedNil, rmlines.ErrItemsNotFound) {
		t.Errorf("a typed nil matches a sentinel")
	}
	if typedNil.Error() == "" {
		t.Errorf("a typed nil spells nothing")
	}
	if errors.Is(err, errors.New("rmlines: items the page does not hold as living ones")) {
		t.Errorf("an error matches a plain error by its text")
	}
}
