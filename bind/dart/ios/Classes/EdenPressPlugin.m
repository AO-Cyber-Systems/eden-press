// Copyright (c) 2026 AO Cyber Systems
// SPDX-License-Identifier: MIT
//
// Placeholder translation unit.
//
// eden_press is an ffiPlugin: Dart calls the Go C ABI directly via dart:ffi,
// so there is no platform-channel plugin class and no Objective-C to write.
// CocoaPods still wants at least one source file for a pod that declares
// source_files, and an empty pod can be pruned from the build — taking the
// vendored xcframework's symbols with it.
//
// The reference below is what makes the linker KEEP the Go archive: iOS uses
// a c-archive that is statically linked into the app binary (which is why
// native_loader.dart calls DynamicLibrary.process() rather than .open()), and
// a static archive contributes nothing unless something references it.
#import <Foundation/Foundation.h>

// Declared by the Go c-archive header (libpress.h) that build-ios.sh packages
// into the xcframework alongside the library.
extern char *PressRender(char *req);

__attribute__((used)) static void *edenPressKeepSymbols[] = {
    (void *)&PressRender,
};
