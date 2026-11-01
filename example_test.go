// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/altessa-s/rmlines-go"
)

func ExampleParse() {
	data, err := os.ReadFile("testdata/v6-samples/Normal_AB.rm")
	if err != nil {
		log.Fatal(err)
	}
	page, err := rmlines.Parse(data)
	switch {
	case errors.Is(err, rmlines.ErrUnsupportedVersion):
		var rmErr *rmlines.Error
		errors.As(err, &rmErr)
		fmt.Println("a page of version", rmErr.Version)
		return
	case err != nil:
		log.Fatal(err)
	}
	if page.Text != nil { // a page of ink alone carries no typed text
		for _, p := range page.Text.Paragraphs {
			fmt.Println(p.Text)
		}
	}
	fmt.Println(len(page.Strokes), "strokes,", page.SkippedBlocks, "blocks lost")
	// Output:
	// AB
	// 0 strokes, 0 blocks lost
}

func ExampleWrite() {
	heading := uint8(rmlines.StyleHeading)
	points := []rmlines.DraftPoint{rmlines.DefaultPoint(0, 100), rmlines.DefaultPoint(100, 100)}
	draft := &rmlines.Draft{
		Author: [16]byte{0x0a, 0x41, 0x8e, 0x1a},
		Sheet:  rmlines.Sheet{ScreenWidth: 1404, ScreenHeight: 1872, CanvasWidth: 1404, CanvasHeight: 1872},
		Column: rmlines.ColumnOf(rmlines.ColumnMedium),
		Paragraphs: []rmlines.DraftParagraph{
			{Text: "Heading", Style: &heading},
			{Text: "Body"},
		},
		Strokes: []rmlines.DraftStroke{rmlines.DefaultStroke(rmlines.ToolFineliner2, rmlines.ColorBlack, points)},
	}
	data, err := rmlines.Write(draft)
	if err != nil {
		log.Fatal(err)
	}
	page, err := rmlines.Parse(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(page.Text.Paragraphs[0].Text, page.Text.Paragraphs[0].Format.HeadingLevel())
	// Output: Heading 1
}

func ExampleEraseItems() {
	data, err := os.ReadFile("testdata/v6-samples/Wikipedia_highlighted_p1.rm")
	if err != nil {
		log.Fatal(err)
	}
	marker, err := rmlines.HighlighterItems(data)
	if err != nil {
		log.Fatal(err)
	}
	edited, err := rmlines.EraseItems(data, marker)
	if err != nil {
		log.Fatal(err)
	}
	page, err := rmlines.Parse(edited)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(marker), "marks erased,", len(page.Highlights), "left")
	// Output: 4 marks erased, 0 left
}

func ExampleParseItemID() {
	id, err := rmlines.ParseItemID("1:1234")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(id.Author, id.Counter, id)
	// Output: 1 1234 1:1234
}

func ExampleSetLogger() {
	rmlines.SetLogger(func(level rmlines.LogLevel, message string) {
		log.Printf("rmlines %v: %s", level, message)
	})
	defer rmlines.SetLogger(nil)
	// What a parse could not read is reported to the function above.
}
