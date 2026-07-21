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

/// The JS-free rendering surface (DART-04): builds a Flutter widget tree
/// directly from `Output.Model`'s schema-v2 [Block]s -- never from
/// `Output.html` (see model.dart's library doc / 07-05-TRD.md anti_patterns,
/// Pitfall 3: presentation MathML carries no TeX annotation and chroma spans
/// are lossy DOM soup, so HTML is not a viable source for these two blocks).
///
/// - `math` blocks render via `flutter_math_fork`'s `Math.tex` from the raw
///   TeX in [Block.text], with [Block.display] selecting [MathStyle.display]
///   vs [MathStyle.text].
/// - `code` blocks render via `flutter_highlighting`'s `HighlightView` from
///   the raw source in [Block.text] and the fenced info-string in
///   [Block.language].
/// - `paragraph`/`heading`/`list`/unknown blocks render as plain Flutter
///   text -- not this TRD's focus, but rendered so the surface is complete.
///
/// No HTML/DOM-parsing package, no webview, no JavaScript is imported or
/// invoked anywhere in this file.
library;

import 'package:flutter/material.dart';
import 'package:flutter_highlighting/flutter_highlighting.dart';
import 'package:flutter_highlighting/themes/github.dart';
import 'package:flutter_math_fork/flutter_math.dart';

import 'model.dart';

/// Default `flutter_highlighting` theme for `code` blocks: GitHub's, chosen
/// for neutral, high-contrast readability. [EdenPressView.highlightTheme]
/// can override it; picking a different default is a presentation decision,
/// not a DART-04 correctness concern.
const Map<String, TextStyle> defaultHighlightTheme = githubTheme;

/// Renders an [Output]'s schema-v2 [ModelDocument] natively, in document
/// order, from its raw [Block] content -- the JS-free surface DART-04 exists
/// to prove.
class EdenPressView extends StatelessWidget {
  const EdenPressView(
    this.output, {
    super.key,
    this.highlightTheme = defaultHighlightTheme,
  });

  /// The decoded render output (typically from [render]/`eden_press.dart`),
  /// carrying the schema-v2 [ModelDocument] this view walks.
  final Output output;

  /// The `flutter_highlighting` theme used for `code` blocks.
  final Map<String, TextStyle> highlightTheme;

  @override
  Widget build(BuildContext context) {
    final blocks = <Widget>[
      for (final section in output.model.sections)
        for (final block in section.blocks) _buildBlock(block),
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: blocks,
    );
  }

  Widget _buildBlock(Block block) {
    switch (block.kind) {
      case BlockKind.math:
        return Math.tex(
          block.text,
          mathStyle: block.display ? MathStyle.display : MathStyle.text,
        );
      case BlockKind.code:
        return HighlightView(
          block.text,
          languageId: block.language,
          theme: highlightTheme,
        );
      case BlockKind.heading:
        return Text(block.text, style: _headingStyle(block.level));
      case BlockKind.list:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            for (final item in block.items)
              Text(
                '${'  ' * item.level}${block.ordered ? '-' : '•'} ${item.text}',
              ),
          ],
        );
      case BlockKind.paragraph:
      case BlockKind.unknown:
        return Text(block.text);
    }
  }

  TextStyle _headingStyle(int level) {
    final double size = switch (level) {
      1 => 28.0,
      2 => 24.0,
      3 => 20.0,
      _ => 18.0,
    };
    return TextStyle(fontSize: size, fontWeight: FontWeight.bold);
  }
}
