#!/usr/bin/env bash
# Copyright (c) 2026 AO Cyber Systems
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#
# SPDX-License-Identifier: MIT

# build-android.sh cross-compiles the DART-01 cgo shim (bind/capi) into ONE
# `libpress.so` per Android ABI via `-buildmode=c-shared`, using the Android
# NDK's own per-triple `<triple><api>-clang` as CC. This is DART-02's native
# Android recipe -- plain `go build`, NO `gomobile bind` anywhere.
#
# Output layout mirrors an Android project's jniLibs directory shape, the exact
# input contract 07-05's plugin_ffi android/ package vendors:
#
#   bind/capi/build/android/jniLibs/<abi>/libpress.so
#   bind/capi/build/android/jniLibs/<abi>/libpress.h
#
# ANDROID_NDK_HOME is REQUIRED. It is a local toolchain install, not a network
# secret -- it is unset on this dev host by design; CI provisions it (see
# .github/workflows/dart-native.yml). When unset, this script fails fast with a
# precise remediation message instead of a cryptic "clang: not found" -- the
# authoritative Android verification is the CI job, not this local run.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ -z "${ANDROID_NDK_HOME:-}" ]; then
	echo "FAIL: ANDROID_NDK_HOME is not set." >&2
	echo "" >&2
	echo "Install the Android NDK, e.g.:" >&2
	echo "  sdkmanager \"ndk;27.2.12479018\"" >&2
	echo "or via Android Studio's SDK Manager, then:" >&2
	echo "  export ANDROID_NDK_HOME=/path/to/Android/sdk/ndk/27.2.12479018" >&2
	echo "" >&2
	echo "CI provisions the NDK automatically (.github/workflows/dart-native.yml);" >&2
	echo "locally this script is best-effort and the CI job is authoritative." >&2
	exit 1
fi

# Resolve the NDK prebuilt toolchain host-tag directory from uname(1).
case "$(uname -s)" in
Darwin) HOST_TAG="darwin-x86_64" ;;
Linux) HOST_TAG="linux-x86_64" ;;
*)
	echo "FAIL: unsupported host OS for NDK cross-compilation: $(uname -s)" >&2
	exit 1
	;;
esac

NDK_BIN="$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/$HOST_TAG/bin"
if [ ! -d "$NDK_BIN" ]; then
	echo "FAIL: expected NDK toolchain directory not found: $NDK_BIN" >&2
	echo "Confirm ANDROID_NDK_HOME points at an actual NDK install root." >&2
	exit 1
fi

OUT_ROOT="bind/capi/build/android/jniLibs"
mkdir -p "$OUT_ROOT"

# build_abi ABI GOARCH CLANG-TRIPLE [GOARM]
#
# GOTCHA: the per-triple clang binary name AND its embedded API-level suffix
# (here: 21) vary by NDK version -- resolve them against the actual installed
# NDK's toolchains/llvm/prebuilt/<host>/bin listing before repinning. API level
# 21 is chosen to match the project's minSdk.
build_abi() {
	abi="$1"
	goarch="$2"
	triple="$3"
	goarm="${4:-}"
	cc="$NDK_BIN/$triple"

	if [ ! -x "$cc" ]; then
		echo "FAIL: expected NDK clang not found or not executable: $cc" >&2
		echo "Listing $NDK_BIN for clang-like binaries to help resolve the correct" >&2
		echo "triple/API suffix for this NDK version:" >&2
		for f in "$NDK_BIN"/*clang*; do
			echo "  $(basename "$f")" >&2
		done
		exit 1
	fi

	out_dir="$OUT_ROOT/$abi"
	mkdir -p "$out_dir"

	if [ -n "$goarm" ]; then
		echo "build-android: $abi (GOARCH=$goarch GOARM=$goarm) via $triple"
		CGO_ENABLED=1 GOOS=android GOARCH="$goarch" GOARM="$goarm" CC="$cc" \
			go build -buildmode=c-shared -o "$out_dir/libpress.so" ./bind/capi
	else
		echo "build-android: $abi (GOARCH=$goarch) via $triple"
		CGO_ENABLED=1 GOOS=android GOARCH="$goarch" CC="$cc" \
			go build -buildmode=c-shared -o "$out_dir/libpress.so" ./bind/capi
	fi

	test -f "$out_dir/libpress.so" || {
		echo "FAIL: $out_dir/libpress.so not produced" >&2
		exit 1
	}
	test -f "$out_dir/libpress.h" || {
		echo "FAIL: $out_dir/libpress.h not produced" >&2
		exit 1
	}

	# Verify the cross-compile actually targeted the expected ABI's arch.
	file_out="$(file "$out_dir/libpress.so")"
	echo "  $file_out"
	case "$abi" in
	arm64-v8a)
		echo "$file_out" | grep -qiE 'aarch64|arm64' || {
			echo "FAIL: $out_dir/libpress.so does not report as arm64/aarch64" >&2
			exit 1
		}
		;;
	armeabi-v7a)
		echo "$file_out" | grep -qiE 'arm' || {
			echo "FAIL: $out_dir/libpress.so does not report as arm" >&2
			exit 1
		}
		;;
	x86_64)
		echo "$file_out" | grep -qiE 'x86-64|x86_64' || {
			echo "FAIL: $out_dir/libpress.so does not report as x86_64" >&2
			exit 1
		}
		;;
	x86)
		echo "$file_out" | grep -qiE '80386|i386|x86' || {
			echo "FAIL: $out_dir/libpress.so does not report as x86/i386" >&2
			exit 1
		}
		;;
	esac
}

# CRITICAL: straight `go build -buildmode=c-shared` per ABI. NO gomobile bind.
build_abi arm64-v8a arm64 aarch64-linux-android21-clang
build_abi armeabi-v7a arm armv7a-linux-androideabi21-clang 7
build_abi x86_64 amd64 x86_64-linux-android21-clang
build_abi x86 386 i686-linux-android21-clang

echo "PASS: all 4 Android ABIs built under $OUT_ROOT/{arm64-v8a,armeabi-v7a,x86_64,x86}; no gomobile used."
