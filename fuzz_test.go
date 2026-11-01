// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

// FuzzParse feeds the parser arbitrary bytes, seeded with every page in
// testdata. It must answer a page or an error and never crash, and a page it
// answers must take an erase of its own first mark.
//
// The erase may refuse with ErrUnverified, its own check that the edited page
// reads back as meant, which arbitrary bytes are free to fail; or with
// ErrOutOfMemory. Any other refusal is a failure: every item the page shows is
// one the editor can erase. The seed in testdata/fuzz/FuzzParse keeps a page
// whose highlight carries both a value and a deleted length.
func FuzzParse(f *testing.F) {
	for _, dir := range []string{"testdata", filepath.Join("testdata", "v6-samples")} {
		matches, err := filepath.Glob(filepath.Join(dir, "*.rm"))
		if err != nil {
			f.Fatal(err)
		}
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(data)
		}
	}
	f.Add([]byte(rmlines.Header))
	f.Fuzz(func(t *testing.T, data []byte) {
		page, err := rmlines.Parse(data)
		if err != nil {
			if page != nil {
				t.Fatalf("a page and an error together")
			}
			return
		}
		_, _ = rmlines.HighlighterItems(data)
		if page.Text != nil {
			for line := range page.Text.Paragraphs {
				page.Text.CheckboxState(line)
			}
		}

		before := livingIDs(page)
		if len(before) == 0 {
			return
		}
		target := before[0]
		edited, err := rmlines.EraseItems(data, []rmlines.ItemID{target})
		switch {
		case errors.Is(err, rmlines.ErrUnverified), errors.Is(err, rmlines.ErrOutOfMemory):
			return
		case err != nil:
			t.Fatalf("erasing %v, which the page named: %v", target, err)
		}
		after, err := rmlines.Parse(edited)
		if err != nil {
			t.Fatalf("the edited page does not parse: %v", err)
		}
		var want []rmlines.ItemID
		for _, id := range before {
			if id != target {
				want = append(want, id)
			}
		}
		if got := livingIDs(after); !slices.Equal(got, want) {
			t.Fatalf("after erasing %v: %v, want %v", target, got, want)
		}
	})
}

// livingIDs answers the identifiers of a page's strokes and then its
// highlights.
func livingIDs(page *rmlines.Page) []rmlines.ItemID {
	var ids []rmlines.ItemID
	for _, s := range page.Strokes {
		ids = append(ids, s.ID)
	}
	for _, h := range page.Highlights {
		ids = append(ids, h.ID)
	}
	return ids
}
