// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/altessa-s/rmlines-go"
)

// FuzzParse feeds the parser arbitrary bytes, seeded with every page in
// testdata. It must answer a page or an error, and never crash; a page it
// answers must take an erase of its own marks.
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
		if len(page.Strokes) > 0 {
			_, _ = rmlines.EraseItems(data, []rmlines.ItemID{page.Strokes[0].ID})
		}
		if page.Text != nil {
			for line := range page.Text.Paragraphs {
				page.Text.CheckboxState(line)
			}
		}
	})
}
