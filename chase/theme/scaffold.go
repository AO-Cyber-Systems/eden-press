// Copyright (c) 2026 AO Cyber Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// SPDX-License-Identifier: MIT

package theme

// scaffold.go retains the reserved ScaffoldThemeName identity a ThemeSet
// registers its internal scaffold theme under (see pack.go's NewThemeSet).
// The scaffold-reset and advanced-background CSS text itself relocated to
// profiles/slides (TRD 02-03, MODEL-04's de-hardcoding move): chase/theme
// now receives that CSS as a caller-supplied parameter to NewThemeSet
// rather than holding it as a package-level constant of its own -- this
// package has no opinion about what a "scaffold" theme's rules actually
// say, only that one reserved identity exists to skip the scaffold-prepend
// step when packing it (Test-list case 8).

// ScaffoldThemeName is the reserved theme name a ThemeSet registers its
// internal scaffold identity under (see pack.go's NewThemeSet), used to
// skip the scaffold-prepend step when packing the scaffold theme itself
// (Test-list case 8).
const ScaffoldThemeName = "scaffold"
