// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines

// CConstants lets the external tests hold the Go constants against the C
// library's.
var CConstants = cConstants

// LogInstalls answers how many times the C log sink was installed.
func LogInstalls() int32 { return logInstalls.Load() }
