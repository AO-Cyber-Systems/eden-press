#!/usr/bin/env bash
# Copyright (c) 2026 AO Cyber Systems
# SPDX-License-Identifier: MIT
#
# Verifies the eden_press Flutter plugin scaffolding stays consistent with the
# native build scripts that feed it.
#
# The failure this exists to prevent is silent: if build-android.sh's output
# directory or build-ios.sh's xcframework name moves, the plugin still BUILDS
# — it just ships without a native library, and the app fails at runtime with
# an opaque dlopen error on a user's device. Nothing else in the repo connects
# those two sides, because the binaries themselves are gitignored and so can
# never be checked in as evidence.
#
# Checks structure and cross-references only; it never needs a built artifact,
# so it runs everywhere including a plain `go test` machine.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

echo "check-dart-native-scaffold: plugin structure"
for f in \
	bind/dart/pubspec.yaml \
	bind/dart/android/build.gradle \
	bind/dart/android/src/main/AndroidManifest.xml \
	bind/dart/ios/eden_press.podspec \
	scripts/vendor-dart-native.sh; do
	[ -f "$f" ] || fail "missing required scaffolding file: $f"
done

# pubspec must declare BOTH platforms as ffiPlugin, or flutter never builds
# the native module at all.
for platform in android ios; do
	grep -A 1 "^      $platform:" bind/dart/pubspec.yaml | grep -q "ffiPlugin: true" ||
		fail "bind/dart/pubspec.yaml does not declare ffiPlugin: true for $platform"
done

echo "check-dart-native-scaffold: android output path agreement"
# build-android.sh's OUT_ROOT must be what vendor-dart-native.sh reads.
grep -q 'OUT_ROOT="bind/capi/build/android/jniLibs"' scripts/build-android.sh ||
	fail "scripts/build-android.sh OUT_ROOT changed; update scripts/vendor-dart-native.sh (ANDROID_SRC)"
grep -q 'ANDROID_SRC="bind/capi/build/android/jniLibs"' scripts/vendor-dart-native.sh ||
	fail "scripts/vendor-dart-native.sh ANDROID_SRC no longer matches build-android.sh's OUT_ROOT"
# vendor destination must be what build.gradle packages.
grep -q 'ANDROID_DST="bind/dart/android/src/main/jniLibs"' scripts/vendor-dart-native.sh ||
	fail "scripts/vendor-dart-native.sh ANDROID_DST changed; update bind/dart/android/build.gradle"
grep -q "main.jniLibs.srcDirs = \['src/main/jniLibs'\]" bind/dart/android/build.gradle ||
	fail "bind/dart/android/build.gradle jniLibs.srcDirs no longer matches vendor-dart-native.sh's ANDROID_DST"

echo "check-dart-native-scaffold: ios artifact-name agreement"
grep -q 'XCFRAMEWORK="\$BUILD_ROOT/EdenPress.xcframework"' scripts/build-ios.sh ||
	fail "scripts/build-ios.sh xcframework name changed; update the podspec + vendor-dart-native.sh"
grep -q 'IOS_DST="bind/dart/ios/EdenPress.xcframework"' scripts/vendor-dart-native.sh ||
	fail "scripts/vendor-dart-native.sh IOS_DST no longer matches build-ios.sh's xcframework name"
grep -q "s.vendored_frameworks = 'EdenPress.xcframework'" bind/dart/ios/eden_press.podspec ||
	fail "podspec vendored_frameworks no longer matches vendor-dart-native.sh's IOS_DST"
# A c-archive payload MUST be a static framework, or the symbols are dropped.
grep -q "s.static_framework = true" bind/dart/ios/eden_press.podspec ||
	fail "podspec must set static_framework = true (the payload is a Go c-archive .a)"

echo "check-dart-native-scaffold: C ABI symbol agreement"
# The Dart lookup name, the Go //export, and the podspec's link-retention
# reference must all name the same symbol.
grep -q '//export PressRender' bind/capi/capi.go ||
	fail "bind/capi no longer exports PressRender"
grep -q "pressRenderSymbol = 'PressRender'" bind/dart/lib/src/ffi_bindings.dart ||
	fail "ffi_bindings.dart's symbol name no longer matches bind/capi's //export"
grep -q 'extern char \*PressRender' bind/dart/ios/Classes/EdenPressPlugin.m ||
	fail "iOS symbol-retention reference no longer names PressRender; the static archive may be stripped"

echo "PASS: dart native scaffolding is consistent with the build scripts."
