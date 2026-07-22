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

// 08-07-TRD.md Test list, driven RED -> GREEN, outermost to innermost. This
// commit covers cases 1-3 (computeFitFontSize, Task 1); cases 4-5 (FitText
// widget + JS-free assertion, Task 2) land in a follow-up commit:
//   1. computeFitFontSize returns maxFontSize when text fits at max in the
//      given constraints (SHRINK-ONLY -- never upscales past max).
//   2. A long heading in a narrow BoxConstraints returns a size STRICTLY
//      LESS than maxFontSize that does not overflow (height <= maxHeight,
//      width <= maxWidth).
//   3. Monotonic: for the same text, a wider constraint yields a fitted size
//      >= a narrower constraint's fitted size.
//
// No generated/property-based data -- every fixture below is a hand-built
// fixed string + explicit BoxConstraints (mirrors render_surface_test.dart's
// testWidgets/unit-test shape).

import 'package:eden_press/src/fit_text.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

const _longHeading =
    'This Is An Extremely Long Heading Title That Would Normally Overflow '
    'A Narrow Slide Column';

void main() {
  group('computeFitFontSize', () {
    test(
      'Case 1: text that already fits at maxFontSize returns maxFontSize '
      '(shrink-only, never upscales)',
      () {
        const style = TextStyle();
        final fitted = computeFitFontSize(
          'Hi',
          const BoxConstraints(maxWidth: 400, maxHeight: 200),
          style: style,
          maxFontSize: 24,
          minFontSize: 8,
        );
        expect(fitted, 24.0);
      },
    );

    test(
      'Case 2: a long heading in narrow constraints returns < maxFontSize '
      'and does not overflow (height/width both within bounds)',
      () {
        const style = TextStyle();
        const constraints = BoxConstraints(maxWidth: 120, maxHeight: 100);

        final fitted = computeFitFontSize(
          _longHeading,
          constraints,
          style: style,
          maxFontSize: 48,
          minFontSize: 8,
        );
        expect(fitted, lessThan(48.0));

        final tp = TextPainter(
          text: TextSpan(
            text: _longHeading,
            style: style.copyWith(fontSize: fitted),
          ),
          textDirection: TextDirection.ltr,
        )..layout(maxWidth: constraints.maxWidth);
        expect(tp.height, lessThanOrEqualTo(constraints.maxHeight));
        expect(tp.width, lessThanOrEqualTo(constraints.maxWidth));
        expect(tp.didExceedMaxLines, isFalse);
      },
    );

    test(
      'Case 3: monotonic -- a wider constraint fits >= a narrower '
      "constraint's fitted size, for the same text",
      () {
        const style = TextStyle();
        final narrow = computeFitFontSize(
          _longHeading,
          const BoxConstraints(maxWidth: 100, maxHeight: 200),
          style: style,
          maxFontSize: 48,
          minFontSize: 8,
        );
        final wide = computeFitFontSize(
          _longHeading,
          const BoxConstraints(maxWidth: 400, maxHeight: 200),
          style: style,
          maxFontSize: 48,
          minFontSize: 8,
        );
        expect(wide, greaterThanOrEqualTo(narrow));
      },
    );
  });
}
