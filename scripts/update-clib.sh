#!/bin/sh
# Copyright 2026 ALTESSA SOLUTIONS INC.
# Use of this source code is governed by an MIT-style license that can be
# found in the LICENSE file.

# Re-vendor rmlines-lib into internal/clib: the library sources and headers,
# its licence, and the tag they were taken at. The tests and tools of the C
# library stay behind; this module tests through its own API.
#
#   scripts/update-clib.sh ../rmlines-lib

set -eu

if [ "$#" -ne 1 ]; then
    echo "usage: $0 <path-to-rmlines-lib>" >&2
    exit 2
fi

source_dir=$1
module_dir=$(cd "$(dirname "$0")/.." && pwd)
clib="$module_dir/internal/clib"

if [ ! -f "$source_dir/include/rmlines/rmlines.h" ]; then
    echo "$source_dir does not look like rmlines-lib" >&2
    exit 1
fi

version=$(git -C "$source_dir" describe --tags --always --dirty)

rm -rf "$clib/src" "$clib/include"
mkdir -p "$clib/src" "$clib/include/rmlines"
cp "$source_dir"/src/*.c "$source_dir"/src/*.h "$clib/src/"
cp "$source_dir"/include/rmlines/*.h "$clib/include/rmlines/"
cp "$source_dir/LICENSE" "$source_dir/COPYRIGHT" "$clib/"
printf '%s\n' "$version" > "$clib/VERSION"

# Each C source compiles through a wrapper in the package root, because cgo
# compiles only the .c files of the package directory itself.
for source in "$clib"/src/*.c; do
    name=$(basename "$source" .c)
    wrapper="$module_dir/clib_${name#rm_}.c"
    if [ ! -f "$wrapper" ]; then
        echo "warning: $name.c has no wrapper; add $wrapper" >&2
    fi
done

echo "vendored rmlines-lib $version"
