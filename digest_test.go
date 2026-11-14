// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package rmlines_test

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// clibDigest answers the digest scripts/update-clib.sh writes into
// clib_digest.h, computed the way the script computes it with find, sort and
// sha256sum: for every regular file under internal/clib, its SHA-256 in hex,
// a space, its path as find prints it from there ("./src/x.c") and a newline,
// the lines in byte order (LC_ALL=C sort), and the SHA-256 of all of them.
// Only the Go toolchain is needed, so it runs on Windows too.
func clibDigest(t *testing.T) string {
	t.Helper()
	const root = "internal/clib"
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// find -type f lists regular files only, not links to them.
		if !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, "./"+filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The script sorts whole lines, each its path; a path holding a newline
	// would break both, and the tree holds none.
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%x %s\n", sha256.Sum256(data), path)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// clib_digest.h records, beside the upstream version the vendored sources
// were taken at, a digest of every file under internal/clib. Holding the
// committed tree to that digest ties it to the recorded version: a hand edit
// or a partial update of the vendored sources fails this test instead of
// shipping unnoticed. scripts/update-clib.sh changes the tree and the digest
// together.
func TestVendoredDigest(t *testing.T) {
	header, err := os.ReadFile("clib_digest.h")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`#define RMLINES_GO_CLIB_DIGEST "%s"`, clibDigest(t))
	if !strings.Contains(string(header), want) {
		t.Fatalf("clib_digest.h does not hold the digest of internal/clib: want %s; re-run scripts/update-clib.sh", want)
	}
}
