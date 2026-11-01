// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

// -std=c2x and not -std=c23: GCC 13 knows the standard only by its draft
// name, while GCC 14 and clang 19 or newer take both. The library spells its
// constants with constexpr, which clang 18 does not read, so no flag rescues
// an older compiler.

/*
#cgo CFLAGS: -std=c2x -I${SRCDIR}/internal/clib/include -I${SRCDIR}/internal/clib/src
#cgo linux LDFLAGS: -lm

#include "rmlines/rmlines.h"

static const char *rmgo_version(void) { return RMLINES_VERSION_STRING; }
static double rmgo_ink_units_per_raw_width(void) { return RM_INK_UNITS_PER_RAW_WIDTH; }
static double rmgo_ink_minimum_stroke_width(void) { return RM_INK_MINIMUM_STROKE_WIDTH; }
static double rmgo_ink_maximum_stroke_width(void) { return RM_INK_MAXIMUM_STROKE_WIDTH; }
static double rmgo_ink_faint_alpha(void) { return RM_INK_FAINT_ALPHA; }
static int rmgo_supported_version(void) { return RM_SUPPORTED_VERSION; }
static const char *rmgo_header(void) { return RM_HEADER; }
*/
import "C"

// Version is the version of the vendored rmlines-lib.
const Version = "1.0.1"

// The scales the format is written in: what a number in the file means
// before anything is drawn.
const (
	// InkUnitsPerRawWidth is what a width in the file is divided by to become
	// a width on the page.
	InkUnitsPerRawWidth = 4.0
	// InkMinimumStrokeWidth keeps thin strokes visible in print; the format's
	// own smallest width is 1.5 units, so it never fires on a width the tablet
	// wrote.
	InkMinimumStrokeWidth = 0.7
	// InkMaximumStrokeWidth guards against a number that is not a width at
	// all; the widest tool is 128 raw, which is 32 units.
	InkMaximumStrokeWidth = 200.0
	// InkFaintAlpha is how opaque a translucent stroke is when the file does
	// not say.
	InkFaintAlpha = 0.25098
)

// SupportedVersion is the one generation of the format this package reads.
const SupportedVersion = 6

// Header is the 43 bytes, padding included, every page of the supported
// version begins with.
const Header = "reMarkable .lines file, version=6          "

// cConstants answers the C library's own spelling of the constants above, so
// that a test can hold the two together.
func cConstants() (version string, header string, supported int, ink [4]float64) {
	return C.GoString(C.rmgo_version()), C.GoString(C.rmgo_header()),
		int(C.rmgo_supported_version()),
		[4]float64{
			float64(C.rmgo_ink_units_per_raw_width()),
			float64(C.rmgo_ink_minimum_stroke_width()),
			float64(C.rmgo_ink_maximum_stroke_width()),
			float64(C.rmgo_ink_faint_alpha()),
		}
}
