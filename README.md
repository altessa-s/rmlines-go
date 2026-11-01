# rmlines-go

Reading and writing the reMarkable `.rm` v6 page format from Go: the page and its strokes, typed text and its paragraphs, highlights and images — and
writing fresh pages the tablet accepts. Nothing here draws, and nothing here has heard of the cloud, the document index or a file manager.

Idiomatic cgo bindings over **[rmlines-lib](https://github.com/altessa-s/rmlines-lib)**, the portable C23 library, whose sources are vendored into
this module: `go get` builds it with no system library to install. It reads the same pages to the same strokes, text and styles as rmlines-lib and
**[rmlines-swift](https://github.com/altessa-s/rmlines-swift)**, and writes the same draft to the same bytes.

Part of **[Slate Drive](https://slatedrive.app)** ([repository](https://github.com/altessa-s/slatedrive)), the macOS app that shows a reMarkable
library in Finder. Its sibling **[rmdoc-swift](https://github.com/altessa-s/rmdoc-swift)** packs and unpacks the `.rmdoc` container these pages
travel in.

## Why it exists

A notebook on disk is a CRDT of tagged blocks, not a picture. Turning one into the other is two jobs: **read the file**, then **draw what was read**.
This module is the first job, for Go: a converter, a sync service or a test harness written in Go reaches the same parser the C and Swift consumers
do, rather than a second reading of the format that drifts from the first.

## Requirements

- Go 1.24 or newer, with cgo enabled (`CGO_ENABLED=1`, the default wherever a C compiler is found).
- A **C23 compiler**: clang 19 or newer, or GCC 13 or newer. Not every compiler that takes a C23 flag is one: the library spells its constants with
  `constexpr`, which clang 18 does not read. Apple's clang in a current Xcode qualifies. MSVC does not, so on Windows that means clang.

The C is compiled with `-std=c2x`, the one spelling GCC 13, GCC 14 and clang 19 all take. Point `CC` at the compiler when the default one is too old:

```sh
CC=clang-19 go build ./...
```

## Installing

```sh
go get github.com/altessa-s/rmlines-go
```

```go
import "github.com/altessa-s/rmlines-go"   // package rmlines
```

## Reading

```go
page, err := rmlines.Parse(data)
switch {
case errors.Is(err, rmlines.ErrUnsupportedVersion):
	var rmErr *rmlines.Error
	errors.As(err, &rmErr)
	fmt.Println("a page of version", rmErr.Version)
case err != nil:
	return err
}
for _, stroke := range page.Strokes {
	fmt.Println(stroke.Tool, stroke.Color, len(stroke.Points))
}
```

Everything `Parse` answers is a Go value. The page is read in C, copied out field by field, and the C memory is released before `Parse` returns:
nothing the caller holds points into it, and there is nothing to close.

Optional values are pointers throughout: `Stroke.ARGB`, `Stroke.Anchor`, `Stroke.SourceID`, `Highlight.Text`, `Paragraph.Format`,
`TextBlock.ColumnWidth`, `Page.Text`, `Page.PaperSize`. Nil means the file did not say, which is a different answer from nought.

Four errors reach this caller, each matched with `errors.Is`:

| Error | When |
|---|---|
| `ErrUnsupportedVersion` | A `.rm` whose header names a generation this reader does not read. `Error.Version` is the number. |
| `ErrUnsupportedFormat` | Bytes that do not begin as a page at all. The first 32 of them travel in `Error.Message`. |
| `ErrUnexpectedEOF` | The stream is shorter than a header, or a block declares a length past the end of the file. |
| `ErrOutOfMemory` | An allocation failed. The whole page is lost, not quietly shortened. |

A block that will not read — a length past the block, a NaN coordinate — does not fail the page. It is stepped over, the loss is added to
`Page.SkippedBlocks`, and a line goes to the logger.

## Writing

```go
heading := uint8(rmlines.StyleHeading)
points := []rmlines.DraftPoint{rmlines.DefaultPoint(0, 100), rmlines.DefaultPoint(100, 100)}
page, err := rmlines.Write(&rmlines.Draft{
	Author: authorUUID, // [16]byte, as the document's .content spells it
	Sheet:  rmlines.Sheet{ScreenWidth: 1404, ScreenHeight: 1872, CanvasWidth: 1404, CanvasHeight: 1872},
	Column: rmlines.ColumnOf(rmlines.ColumnMedium),
	Paragraphs: []rmlines.DraftParagraph{
		{Text: "Heading", Style: &heading},
		{Text: "Body"},
	},
	Strokes: []rmlines.DraftStroke{rmlines.DefaultStroke(rmlines.ToolFineliner2, rmlines.ColorBlack, points)},
})
if err != nil {
	return err
}
return os.WriteFile("page.rm", page, 0o644)
```

`DefaultPoint` and `DefaultStroke` fill in the defaults a point and a stroke usually carry, as the C library does. Text is UTF-8; ill-formed bytes are
written as U+FFFD, and every count and anchor is taken from the text as written. The same draft writes the same bytes, every time.

The writer builds **fresh pages only**. It never edits an existing file; that contract is the editor's, below. Two refusals, and both mean nothing was
written:

- `ErrAnchorOutOfRange` — ink fastened to a paragraph the draft has not got. Written anyway it would hang on nothing and vanish without a trace.
- `ErrTooManyToNumber` — more items of one kind than a page's identifier ranges can name; `Error.Message` says of what.

A paragraph or an image file name holding a NUL byte is refused with `ErrInvalidArgument`: the C library reads them as C strings, and a NUL would cut
them short in silence.

## Editing

```go
marker, err := rmlines.HighlighterItems(data)   // every living mark of the highlighter
if err != nil {
	return err
}
edited, err := rmlines.EraseItems(data, marker)
if err != nil {
	return err
}
return os.WriteFile("page.rm", edited, 0o644)
```

Every parsed stroke and highlight carries the identifier the page calls it by (`ItemID`, spelled `author:counter` by `String` and read back by
`ParseItemID`), and that is the handle an edit finds it by. `EraseItems` erases the way the tablet's own eraser does: the item stays as a tombstone,
and every other byte goes out as it came in. The result is read back before it is returned, and the refusals mean nothing was written:

- `ErrItemsNotFound` — an identifier is not a living stroke or highlight of this page. The edit does not guess which items were meant.
- `ErrUnverified` — the edited page did not parse back to what the original held less the erased items.

`HighlighterItems` finds the marker on hidden layers too, since showing the layer again would bring it back. A page with a block that would not read
is refused with `ErrIncompletePage` rather than answered in part.

## Diagnostics

```go
rmlines.SetLogger(func(level rmlines.LogLevel, message string) {
	slog.Debug("rmlines", "level", level, "message", message)
})
defer rmlines.SetLogger(nil)
```

Silence until a logger is set, and silence again after `SetLogger(nil)`. The logger is called on whichever goroutine is parsing, so one that touches
shared state guards it itself; it may be replaced at any time, from any goroutine. Every value in every message is a count, a block type or a raw style
number: the page's writing, its text and its file names never reach it.

## Design

**Values, not handles.** A parsed page is plain Go data. Copying it out costs one pass over the page; in return no finaliser, no `Close` and no
lifetime rule reaches the caller, and a page can be kept and shared between goroutines like any other value.

**C memory for C pointers.** A draft is a graph of pointers, and cgo forbids handing C Go memory that holds Go pointers. `Write` builds the whole draft
in C memory and frees it as one before it returns; the tests run clean under `GOEXPERIMENT=cgocheck2`.

**No state between calls.** Each parse, write and erase works in an arena of its own, so any number of them may run at once. The logger is the one
process-wide setting, held atomically on both sides.

**The whole contract.** Every function rmlines-lib exports is reachable: the predicates on tools, styles and formats, the checkbox state of a line,
the column helpers and the header version call into C rather than restating the tablet's numbers in a second place.

**Read, do not draw.** `InkUnitsPerRawWidth` says what a number in the file means (raw width ÷ 4 is a width on the page). Palettes, blend modes, type
metrics and templates belong to a renderer.

## Layout

| Path | What is in it |
|---|---|
| `*.go` | The API: `page.go` (the model and `Parse`), `write.go`, `edit.go`, `itemid.go`, `enums.go`, `errors.go`, `log.go` |
| `clib_*.c` | One line each, compiling a vendored C source into the package: cgo builds only the C files of the package directory |
| `log_bridge.c` | The C sink that hands diagnostic lines to Go |
| `internal/clib/` | The vendored rmlines-lib: `src/`, `include/rmlines/`, its `LICENSE` and the tag it was taken at in `VERSION` |
| `scripts/update-clib.sh` | Re-vendors rmlines-lib from a checkout: `scripts/update-clib.sh ../rmlines-lib` |
| `testdata/` | Pages written on the tablet, and a third-party sample corpus under `v6-samples/` |

```sh
go test -race ./...
go test -run='^$' -fuzz=FuzzParse -fuzztime=30s .
```

## See also

- **[rmlines-lib](https://github.com/altessa-s/rmlines-lib)** — the C23 library these bindings compile, with a `rmdump` tool.
- **[rmlines-swift](https://github.com/altessa-s/rmlines-swift)** — the same format as a Swift package for macOS and iOS.
- **[rmdoc-swift](https://github.com/altessa-s/rmdoc-swift)** — the `.rmdoc` container a document and its pages are exported and imported in.
- **[Slate Drive](https://slatedrive.app)** ([repository](https://github.com/altessa-s/slatedrive)) — the app these packages are part of.

## Acknowledgements

The `.rm` format is not published by reMarkable. This module is built on openly available information and on the research of third-party developers
who documented the format, among them [rmscene](https://github.com/ricklupton/rmscene) and
[remarkable_rust](https://github.com/hiteshjoshi/remarkable_rust).

## Licence

MIT — see [LICENSE](LICENSE). The vendored C under `internal/clib/` is rmlines-lib, under its own MIT licence beside it. The sample corpus under
`testdata/v6-samples/` comes from [remarkable_rust](https://github.com/hiteshjoshi/remarkable_rust) and carries its own licence beside it.

reMarkable is a trademark of reMarkable AS. This project is not affiliated with, endorsed by or sponsored by reMarkable AS.
