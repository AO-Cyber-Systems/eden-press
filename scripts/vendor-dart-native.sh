#!/usr/bin/env bash
# Copyright (c) 2026 AO Cyber Systems
# SPDX-License-Identifier: MIT
#
# Copies the native build products into the eden_press Flutter plugin, so a
# consuming Flutter app can `flutter pub get` the package and have the Go core
# actually link.
#
#   scripts/build-android.sh -> bind/capi/build/android/jniLibs/<abi>/libpress.so
#                            -> bind/dart/android/src/main/jniLibs/<abi>/
#
#   scripts/build-ios.sh     -> bind/capi/build/ios/EdenPress.xcframework
#                            -> bind/dart/ios/EdenPress.xcframework
#
# Neither artifact is committed (.gitignore excludes *.so / *.a /
# *.xcframework/), so this runs after a build, on the machine that built.
#
# Vendoring one platform without the other is normal and supported: iOS
# artifacts cannot be produced off macOS at all, so a Linux CI job legitimately
# vendors Android only. Missing sources are reported and skipped, never fatal;
# the exit code is non-zero only when NEITHER platform had anything to vendor,
# which almost always means the build step was skipped by mistake.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

ANDROID_SRC="bind/capi/build/android/jniLibs"
ANDROID_DST="bind/dart/android/src/main/jniLibs"
IOS_SRC="bind/capi/build/ios/EdenPress.xcframework"
IOS_DST="bind/dart/ios/EdenPress.xcframework"

vendored=0

if [ -d "$ANDROID_SRC" ]; then
	# Replace per-ABI rather than wiping the whole tree: the directory carries
	# a committed README.md that keeps it present in git.
	for abi_dir in "$ANDROID_SRC"/*/; do
		abi="$(basename "$abi_dir")"
		[ -f "$abi_dir/libpress.so" ] || continue
		mkdir -p "$ANDROID_DST/$abi"
		cp "$abi_dir/libpress.so" "$ANDROID_DST/$abi/libpress.so"
		echo "vendor-dart-native: android $abi -> $ANDROID_DST/$abi/libpress.so"
		vendored=$((vendored + 1))
	done
else
	echo "vendor-dart-native: SKIP android — $ANDROID_SRC not present (run scripts/build-android.sh)" >&2
fi

if [ -d "$IOS_SRC" ]; then
	rm -rf "$IOS_DST"
	cp -R "$IOS_SRC" "$IOS_DST"
	echo "vendor-dart-native: ios -> $IOS_DST"
	vendored=$((vendored + 1))
else
	echo "vendor-dart-native: SKIP ios — $IOS_SRC not present (run scripts/build-ios.sh on macOS)" >&2
fi

if [ "$vendored" -eq 0 ]; then
	echo "FAIL: nothing vendored. Build at least one platform first:" >&2
	echo "  scripts/build-android.sh   (needs ANDROID_NDK_HOME)" >&2
	echo "  scripts/build-ios.sh       (needs macOS + Xcode)" >&2
	exit 1
fi

echo "PASS: vendored $vendored native artifact set(s) into bind/dart."
