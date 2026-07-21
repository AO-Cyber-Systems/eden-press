# Objective 6: convert/pptx (native OOXML) - Research

**Researched:** 2026-07-21
**Domain:** OOXML / PresentationML (ECMA-376 / ISO-IEC 29500) hand-rolled PPTX writer using Go stdlib `archive/zip` + `encoding/xml`
**Confidence:** HIGH for OOXML structural facts (cross-verified: Microsoft Learn ISO/IEC 29500-1 API reference + officeopenxml.com via WebSearch + python-pptx's own packaging docs). MEDIUM for Go `archive/zip` determinism specifics (WebSearch cross-verified against pkg.go.dev + community write-ups, not a single canonical stdlib doc page). HIGH for the "no new permissive Go PPTX library" negative claim (re-checked this session against 2026-07-20 STACK.md baseline, identical result).

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| EXP-03 | PPTX via a hand-rolled OOXML writer (`archive/zip` + `encoding/xml`; reject `unioffice`), consuming the docmodel directly (no Chrome), targeting editable text boxes | Standard Stack (stdlib-only confirmed, licensing re-confirmed), Architecture Patterns (zip part layout, `p:sp` shape mapping, group-shape pattern, notes-slide pattern), Common Pitfalls (EMU/chOff trap, clrMap/fmtScheme traps, determinism trap), Open Questions (critical: current `chase/model.Section` has no body-content field — see below) |
</phase_requirements>

## Summary

A minimal, PowerPoint-and-LibreOffice-openable `.pptx` is a ZIP package of ~10-15 wired-together XML parts (content-types manifest, package rels, `presentation.xml`, one slide master/layout/theme triple, N slides, optional notes). None of this requires a third-party library: Go's stdlib `archive/zip` + `encoding/xml` (or even `text/template` for the XML) is sufficient and is the ONLY viable path given licensing — `unioffice` and every fork of it remain AGPLv3/commercial with a network-checked license key, and a fresh search today (2026-07-21) surfaced no new permissive alternative. This re-confirms STACK.md item #10 and PITFALLS.md Pitfall 8 exactly as previously researched; nothing has changed.

The OOXML mechanics needed (zip layout, `<p:sp>` text-box shape XML, EMU math, `chOff`/`chExt` group semantics, notes-slide wiring, determinism, CI verification) are all well-documented, spec-stable, and low-risk to implement. **The real risk in this objective is NOT the OOXML writer — it's the docmodel.** `chase/model.Section` (as merged in Objective 2 / MODEL-01, confirmed by direct inspection of `chase/model/document.go`) deliberately has NO field for paragraph or list body content. Its doc comment states this explicitly: *"Blocks/HTML content are deliberately NOT part of this shape... a field with no consumer in this TRD's Test list is a speculative superset and does not belong here."* The only text content Objective 6 can pull out of `press.Output.Model` is: (a) heading text via `Document.Outline` (grouped by `SectionID`), and (b) speaker notes via `Section.Notes`. There is currently no way to recover ordinary paragraph or list body content from the docmodel as implemented. This is a scope-defining fact the planner must resolve explicitly (see Open Questions #1) — it is not a gap in this research, it is a gap in the upstream data available to consume.

**Primary recommendation:** Build the OOXML writer as a stdlib-only, from-scratch package (`convert/pptx`) that maps one `chase/model.Section` → one slide, with the section's `Outline` entries becoming title/heading text-box shapes and `Section.Notes` becoming the slide's notes part; explicitly re-scope (or explicitly flag as blocked-on-a-model-change) the "paragraphs, lists" portion of EXP-03's shape-mapping ask, since the current docmodel cannot supply that content. Sequence the TRDs static-deck-first (prove openability end-to-end on a trivial fixture before adding EMU utilities, notes, and grouped shapes), and build the EMU-conversion utility and CI verification harness as independently testable units from the start.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `archive/zip` | Go stdlib (1.21+, no material change through 1.25) | Builds the `.pptx` ZIP package (OPC container) | Only dependency-free way to produce a ZIP with full control over per-file `Method`/`Modified` for determinism; PPTX is literally an Open Packaging Conventions ZIP |
| `encoding/xml` | Go stdlib | Marshals/unmarshals the OOXML parts | Sufficient for OOXML's namespaced-but-flat-ish shape XML; avoids a schema-generation dependency |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `text/template` (stdlib, optional) | n/a | Alternative to `encoding/xml` struct-marshaling for the boilerplate parts (`[Content_Types].xml`, `theme1.xml`, `slideMaster1.xml`) that never vary per-deck | Use if struct-tag-driven `encoding/xml` marshaling of deeply nested DrawingML (`a:xfrm`, `a:off`, `a:ext`, `a:chOff`, `a:chExt`) proves awkward — many hand-rolled OOXML writers in other languages (python-pptx's internals, PptxGenJS) use literal XML string templates for the static parts and only programmatically build the per-slide/per-shape parts |
| LibreOffice (`soffice`, headless) | any recent stable (7.x+) | CI-only, black-box "does an independent OOXML consumer open this file" smoke test | Only needed in CI/test tooling, never a runtime/production dependency of the writer itself |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `archive/zip`+`encoding/xml` | `unioffice` (`unidoc/unioffice`) or any of its forks (`5andr0/unioffice`, `AlexGames73/unioffice-free`, `hakwolf/unioffice`, `jkandasa/unioffice`) | REJECTED. AGPLv3 + commercial-license-key tier that requires a metered network check-in even on the "free" tier — directly conflicts with Eden Press's MIT license and no-implicit-network posture. Re-confirmed 2026-07-21: no license/business-model change since 2026-07-20 baseline, and no new permissive competitor has appeared. |
| Hand-rolled `encoding/xml` structs | Literal XML string templates (`text/template` or raw `fmt.Sprintf`) for static/boilerplate parts | Not mutually exclusive — see Supporting table. Pure string templates are more fragile to XML-escaping bugs (must manually escape user text via `text/template`'s `html/template`-style escaping or `xml.EscapeText`); struct-based marshaling gets escaping for free but is more verbose for deeply nested DrawingML. A pragmatic split (templates for invariant boilerplate parts, structs for per-slide/per-shape content where escaping matters) is reasonable. |

**Installation:**
```bash
# none — stdlib only, no third-party dependency to add
```

## Architecture Patterns

### Recommended Project Structure
```
convert/pptx/
├── pptx.go          # public entry point: ToPPTX(model.Document, Options) ([]byte, error)
├── emu.go           # EMU-conversion utility (Inches/Points/Centimeters -> int64 EMU; slide-size constants)
├── emu_test.go       # independently unit-tested per Objective 6 success criteria
├── package.go       # zip-part orchestration: deterministic ordering, fixed timestamps, Content-Types + rels wiring
├── contenttypes.go  # [Content_Types].xml builder
├── shapes.go        # p:sp text-box + p:grpSp group-shape XML builders
├── slide.go         # one chase/model.Section -> one ppt/slides/slideN.xml
├── notes.go         # Section.Notes -> ppt/notesSlides/notesSlideN.xml (skipped when empty)
├── theme.go         # static theme1.xml (fixed clrScheme/fontScheme/fmtScheme)
├── master.go        # static slideMaster1.xml / slideLayout1.xml (with mandatory p:clrMap)
└── testdata/        # fixture .pptx byte-diffs and/or LibreOffice-headless smoke fixtures
```

### Pattern 1: Minimal OPC Part Graph
**What:** A `.pptx` is a ZIP (Open Packaging Conventions container) of XML parts wired together by `.rels` relationship files, plus a package-wide content-type manifest. The minimal set needed for a deck that opens **editably** (not just "repairs and opens") in both PowerPoint and LibreOffice:

```
[Content_Types].xml                              # manifest: Default (by extension) + Override (by part) entries for every part below
_rels/.rels                                      # root package rels -> ppt/presentation.xml (officeDocument), docProps/core.xml, docProps/app.xml
docProps/core.xml                                 # Dublin-Core metadata (title/creator/etc) — PowerPoint is lenient if absent, but include for compatibility
docProps/app.xml                                  # PowerPoint-specific app metadata (slide count etc) — same leniency caveat
ppt/presentation.xml                              # root part: sldMasterIdLst, sldIdLst, sldSz, notesSz
ppt/_rels/presentation.xml.rels                   # -> slideMaster1.xml, each slideN.xml, theme1.xml, presProps.xml, viewProps.xml, tableStyles.xml
ppt/presProps.xml                                 # presentation-level view/print properties — "boring but expected" part
ppt/viewProps.xml                                 # editor view properties — same
ppt/tableStyles.xml                               # empty/default table-style list — same
ppt/theme/theme1.xml                              # clrScheme (12 colors) + fontScheme + fmtScheme (see Pitfall below)
ppt/slideMasters/slideMaster1.xml                 # REQUIRES <p:clrMap> with all 12 attrs (see Pitfall below)
ppt/slideMasters/_rels/slideMaster1.xml.rels       # -> slideLayout1.xml, theme1.xml
ppt/slideLayouts/slideLayout1.xml                 # at minimum one layout (e.g. "Title and Content" or blank)
ppt/slideLayouts/_rels/slideLayout1.xml.rels       # -> slideMaster1.xml
ppt/slides/slideN.xml                             # one per chase/model.Section
ppt/slides/_rels/slideN.xml.rels                   # -> slideLayout1.xml, notesSlideN.xml (only if that section has notes)
ppt/notesMasters/notesMaster1.xml                 # ONLY needed if any slide has notes
ppt/notesMasters/_rels/notesMaster1.xml.rels       # -> theme1.xml
ppt/notesSlides/notesSlideN.xml                    # ONLY for slides with non-empty Section.Notes
ppt/notesSlides/_rels/notesSlideN.xml.rels          # -> slideN.xml, notesMaster1.xml
```

**Confidence:** HIGH. Cross-verified against Microsoft Learn's "Structure of a PresentationML document" reference and python-pptx's own packaging model (which documents the identical minimal part set it writes).

**Key compatibility note:** PowerPoint is lenient — it will silently "repair" and still open a deck missing `presProps.xml`/`viewProps.xml`/`tableStyles.xml`/`docProps/*`, which can mask a real bug during dev. LibreOffice tends to be stricter about **relationship-graph completeness** (every `r:id` must resolve to a real relationship, every relationship target must exist) but more forgiving about missing "nice-to-have" parts. Recommendation: include the full "boring but complete" part set from the very first TRD rather than adding parts reactively after a compatibility bug report (see Common Pitfalls, "boring parts" trap).

### Pattern 2: `<p:sp>` Text-Box Shape → `chase/model` Mapping

**Shape XML shape** (illustrative structure, not literal Go code):
```xml
<p:sp>
  <p:nvSpPr>
    <p:cNvPr id="2" name="Title 1"/>
    <p:cNvSpPr txBox="1"/>
    <p:nvPr/>
  </p:nvSpPr>
  <p:spPr>
    <a:xfrm>
      <a:off x="838200" y="365760"/>
      <a:ext cx="7772400" cy="1143000"/>
    </a:xfrm>
    <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
  </p:spPr>
  <p:txBody>
    <a:bodyPr/>
    <a:lstStyle/>
    <a:p>
      <a:r>
        <a:rPr lang="en-US" sz="4400" b="1"/>
        <a:t>Heading text goes here</a:t>
      </a:r>
    </a:p>
  </p:txBody>
</p:sp>
```
Bullet/list-level paragraphs use `<a:pPr lvl="N" marL="…" indent="…"><a:buChar char="•"/></a:pPr>` (or `<a:buAutoNum type="arabicPeriod"/>` for numbered lists, or `<a:buNone/>` to suppress inherited bullets) as the first child of `<a:p>`, before any `<a:r>`.

**Mapping to `chase/model` (as the model is ACTUALLY implemented today — see Open Questions #1 for the gap):**
- One slide per `Document.Sections[i]`.
- Title/heading shape text = `Document.Outline` entries whose `SectionID == Sections[i].ID` (there can be more than one heading per section if the source markdown has multiple headings inside one section — lowest `Level` becomes the title shape, any additional entries become secondary heading text-box shapes or bullet lines within a body shape).
- `Section.Notes` (`[]string`) → one `<a:p>` per note string in the slide's `notesSlideN.xml` body placeholder (see Pattern 4).
- `Section.Attrs` (`map[string]string`, directive-derived) → NOT currently a designed contract for shape styling; treat any use of these as opportunistic (e.g. a `background`/`class` key, if `chase/directive` is confirmed to populate one) rather than a guaranteed mapping.
- There is currently **no** source for ordinary paragraph or list body text — see Open Questions #1.

**Confidence:** HIGH for the XML shape (matches ECMA-376 DrawingML `CT_Shape`/`CT_TextBody` schema as described by Microsoft Learn and officeopenxml.com). HIGH for the "what's actually in chase/model" claim (verified directly from `chase/model/document.go`, not inferred).

### Pattern 3: EMU Math and Slide-Size Constants

- 1 inch = 914,400 EMU
- 1 centimeter = 360,000 EMU
- 1 millimeter = 36,000 EMU
- 1 point = 12,700 EMU
- Font size (`sz` attribute on `<a:rPr>`/`<a:defRPr>`) is in **hundredths of a point** (centipoints), NOT EMU — `sz="4400"` means 44pt.

Slide-size EMU values (`<p:sldSz cx="…" cy="…" type="…"/>` in `presentation.xml`):
| Aspect | cx (EMU) | cy (EMU) | Inches | `type` attribute |
|--------|----------|----------|--------|-------------------|
| 16:9 widescreen | 12,192,000 | 6,858,000 | 13.333 × 7.5 | `screen16x9` |
| 4:3 standard | 9,144,000 | 6,858,000 | 10 × 7.5 | `screen4x3` |

The `type` attribute is a PowerPoint UI hint (affects the "Design > Slide Size" dropdown selection) — it does not by itself change rendering; `cx`/`cy` are authoritative. Recommended to still set it correctly for round-trip fidelity if a user resizes in PowerPoint's UI later.

**Confidence:** HIGH — these are fixed ECMA-376 constants, cross-verified across Microsoft Learn, officeopenxml.com, and multiple independent OOXML-writing library docs (python-pptx, PptxGenJS); no ambiguity found.

### Pattern 4: Grouped Shapes — `chOff`/`chExt` Semantics

```xml
<p:grpSp>
  <p:grpSpPr>
    <a:xfrm>
      <a:off x="838200" y="1524000"/>       <!-- group's position in the SLIDE's coordinate space -->
      <a:ext cx="4572000" cy="2286000"/>    <!-- group's size in the SLIDE's coordinate space -->
      <a:chOff x="0" y="0"/>                <!-- origin of the CHILD (internal) coordinate space -->
      <a:chExt cx="4572000" cy="2286000"/>  <!-- extent of the CHILD (internal) coordinate space -->
    </a:xfrm>
  </p:grpSpPr>
  <!-- child p:sp / p:pic / p:grpSp elements, each with their OWN <a:xfrm><a:off/><a:ext/></a:xfrm>
       expressed in the CHILD coordinate space defined by chOff/chExt above -->
</p:grpSp>
```

Effective transform for a child shape (child-space `off`/`ext`) into slide-space:
```
scaleX  = ext.cx  / chExt.cx
scaleY  = ext.cy  / chExt.cy
slideX  = off.x + (child.off.x - chOff.x) * scaleX
slideY  = off.y + (child.off.y - chOff.y) * scaleY
slideCX = child.ext.cx * scaleX
slideCY = child.ext.cy * scaleY
```

**Recommended v1 simplification:** set `chOff == off` and `chExt == ext` (i.e., child coordinate space is identical to the group's own slide-space placement, scale = 1, translate = 0). Child shapes' `off`/`ext` values then ARE literal slide-EMU coordinates with zero extra math — this still produces a syntactically and semantically valid, spec-conformant grouped shape (satisfying the "at least one grouped-shape case" success criterion) while eliminating an entire class of scaling-math bugs from the first implementation. A later TRD can introduce true child-space scaling (e.g. to support resizing an entire group by only changing its own `off`/`ext` without touching every child) once the simple case is proven in CI and in real PowerPoint/LibreOffice opens.

**Confidence:** HIGH for the schema shape and the scale/translate formula derivation (ISO/IEC 29500-1's own `CT_GroupTransform2D` description confirms `chOff`/`chExt` establish the children's coordinate system independent of the group's own on-slide position/size). MEDIUM-sourced via WebSearch-mediated officeopenxml.com content (direct `WebFetch` to officeopenxml.com failed with a TLS handshake error this session — content was obtained via WebSearch result summaries instead, cross-checked against Microsoft Learn's ISO/IEC 29500-1 API reference which WAS fetched directly and independently confirms the same semantics).

### Anti-Patterns to Avoid
- **Treating `docProps/*`, `presProps.xml`, `viewProps.xml`, `tableStyles.xml` as safely-skippable:** PowerPoint's silent-repair leniency hides real gaps that a stricter consumer (LibreOffice, a validator, a different PowerPoint build) may reject or flag. Include the full boilerplate part set from day one.
- **Reusing a shared LibreOffice user-profile directory across parallel CI runs:** causes profile-lock hangs. Always pass a unique `-env:UserInstallation=file:///…` per invocation.
- **Applying non-trivial `chOff`/`chExt` scaling before proving the identity (`chOff==off`, `chExt==ext`) case works in both PowerPoint and LibreOffice.**
- **Using `zip.Deflate` for byte-for-byte determinism assertions in tests:** Go's `compress/flate` output has historically been stable across versions but is not a documented cross-version byte-stability guarantee; prefer `zip.Store` for any test that asserts exact output bytes/hashes, reserve `Deflate` for production file-size optimization only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PPTX generation as a whole (packaging semantics, relationship-ID bookkeeping, content-type manifest correctness) | A from-scratch understanding of OPC re-derived from first principles under deadline pressure | The documented minimal part graph in Pattern 1 above, referenced directly against ECMA-376/ISO-29500 and cross-checked against a mature reference implementation's own packaging docs (python-pptx) | OPC has enough interlocking required-attribute traps (see Common Pitfalls) that re-deriving it from scratch without a reference is where most hand-rolled OOXML writers in any language lose the most time |
| A full third-party PPTX library dependency | `unioffice` or any fork | Hand-rolled `archive/zip` + `encoding/xml` | AGPLv3 + commercial license-key/network check-in conflicts with Eden Press's MIT/no-implicit-network posture; confirmed again this session, no alternative has appeared |
| OOXML schema validation | A hand-written subset-schema checker | LibreOffice-headless convert-to-pdf as the practical "does a real OOXML consumer accept this" proxy, optionally supplemented later by the actual ECMA-376/ISO-29500 XSD schema set for stricter validation | Writing a partial/approximate schema validator risks false confidence (passes your own incomplete rules, still fails in real PowerPoint); an independent, spec-conformant consumer is a stronger signal for the effort involved |

**Key insight:** The temptation in this domain is to hand-roll the OOXML *packaging/schema* knowledge from memory or first principles. That's the wrong thing to hand-roll — the writer code itself (zip assembly, XML marshaling) is legitimately hand-rolled per this objective's decision gate, but the *shape of the parts* should be taken directly from the spec/reference dissections documented here, not reinvented.

## Common Pitfalls

### Pitfall 1: EMU/`chOff`/`chExt` group-shape coordinate trap
**What goes wrong:** Child shapes inside a `<p:grpSp>` are placed using the wrong coordinate space (slide-space instead of child-space), producing shapes that render in the wrong place or with the wrong scale, or a `chOff`/`chExt` that doesn't match `off`/`ext`'s aspect ratio, distorting children.
**Why it happens:** The two-coordinate-space design (group's own on-slide placement vs. children's internal placement) is not obvious from the element names alone.
**How to avoid:** Use the `chOff==off`/`chExt==ext` identity simplification for the first implementation (Pattern 4); write the scale/translate formula as its own tested utility before ever emitting a non-identity group.
**Warning signs:** Grouped shapes render correctly in one viewer (e.g. PowerPoint, which is sometimes more forgiving) but wrong in another (LibreOffice) — a sign the coordinate-space math is subtly wrong rather than the XML being malformed.

### Pitfall 2: `slideMaster1.xml`'s mandatory `<p:clrMap>` (all 12 attributes required)
**What goes wrong:** Omitting any of the 12 required color-map attributes (`bg1, tx1, bg2, tx2, accent1-6, hlink, folHlink`) on `<p:clrMap>` produces a file that PowerPoint may silently "repair" (masking the bug) or that a stricter consumer rejects outright.
**Why it happens:** It's easy to assume a partial/default color map is acceptable since most decks don't customize all 12 slots.
**How to avoid:** Always emit the full, fixed 12-attribute `<p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlk"/>` (the canonical identity mapping) as static boilerplate — there is no reason for a hand-rolled writer's v1 to vary this.
**Warning signs:** PowerPoint reports "found a problem and repaired" on open even though the file "looked fine" in a text editor.

### Pitfall 3: `theme1.xml`'s `fmtScheme` requires exactly 3 entries per list
**What goes wrong:** `<a:fmtScheme>`'s child lists (`fillStyleLst`, `lnStyleLst`, `effectStyleLst`, `bgFillStyleLst`) each require **exactly 3** child entries per the ECMA-376 schema (`CT_StyleMatrix`-style fixed cardinality) — a writer that emits 1 or 2 (or a variable number) produces a file that some consumers reject as malformed even though it "looks like valid XML."
**Why it happens:** The 3-entry requirement is a schema cardinality constraint, not obvious from casual inspection of a real .pptx's theme XML unless you count.
**How to avoid:** Treat `theme1.xml` as fully static, hand-copied-once boilerplate (fixed 12-color `clrScheme`, `majorFont`/`minorFont` `fontScheme`, and a `fmtScheme` with exactly 3 entries in each of the 4 lists) — there is no v1 reason to generate this dynamically per-deck.
**Warning signs:** A file opens in PowerPoint (lenient) but fails in a stricter validator or a different consumer.

### Pitfall 4: Determinism vs. Go zip internals
**What goes wrong:** Byte-for-byte non-reproducible output between test runs (breaks golden-file/hash-based tests) due to `zip.FileHeader.Modified` defaulting to "now," and potential (though historically rare) `compress/flate` output drift across Go toolchain versions when using `zip.Deflate`.
**Why it happens:** `archive/zip`'s ergonomic defaults favor "just works" over determinism; nothing in the stdlib API surfaces this as a footgun.
**How to avoid:** Explicitly set `FileHeader.Modified` to a fixed constant (any fixed date at/after the zip format's 1980 floor) on every entry; use `zip.Store` for any test asserting exact bytes/hashes; always add parts in a fixed, explicit order (never range over a Go map for part names).
**Warning signs:** Flaky "golden file" test failures that only reproduce on CI or a different developer machine/Go version.

### Pitfall 5: The docmodel doesn't have what the shape-mapping task assumes (cross-objective risk, not a pure OOXML pitfall)
**What goes wrong:** Attempting to plan/implement "map headings, paragraphs, and lists into shapes" against `chase/model.Section` will stall or force an ad-hoc, undocumented reverse-engineering of body content from somewhere it isn't meant to live, because `Section` has no body-content field.
**Why it happens:** The objective's own framing (and the literal EXP-03 requirement text) references "headings, paragraphs, lists," which was reasonable to assume before `chase/model` was actually implemented; the real, merged `chase/model.Section` (Objective 2 / MODEL-01) deliberately scoped this out.
**How to avoid:** See Open Questions #1 — resolve explicitly at planning time whether Objective 6 re-scopes to headings+notes only, or declares an explicit new dependency on a docmodel extension.
**Warning signs:** A TRD task description that says "render Section body text" without a concrete field reference into `chase/model/document.go`.

## Code Examples

Illustrative OOXML structure only (per this objective's "no code" research directive — these are target-file-format examples, not proposed Go implementation code):

### Content-Types manifest shape (illustrative)
```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
  <Override PartName="/ppt/notesMasters/notesMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesMaster+xml"/>
  <Override PartName="/ppt/notesSlides/notesSlide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>
```

### `presentation.xml` skeleton (illustrative)
```xml
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
                 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
                 xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rIdMaster1"/></p:sldMasterIdLst>
  <p:sldIdLst>
    <p:sldId id="256" r:id="rIdSlide1"/>
    <p:sldId id="257" r:id="rIdSlide2"/>
  </p:sldIdLst>
  <p:sldSz cx="12192000" cy="6858000" type="screen16x9"/>
  <p:notesSz cx="6858000" cy="9144000"/>
</p:presentation>
```

### Notes-slide body placeholder (illustrative)
```xml
<p:notes>
  <p:cSld>
    <p:spTree>
      <!-- nvGrpSpPr / grpSpPr omitted for brevity -->
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Notes Placeholder"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph type="body" idx="1"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr/>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p><a:r><a:t>Speaker note text from Section.Notes[0]</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:notes>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| Screenshot-per-slide PPTX export (Marp's default, and most Chrome-driven pipelines) | Native OOXML with real editable `<p:sp>` text boxes | N/A — this is a deliberate Eden Press differentiator over the Marp/Chrome norm, not an industry-wide shift | Recipient can actually edit text in PowerPoint/LibreOffice instead of getting an unmodifiable image per slide |
| `unioffice` as "the" Go OOXML library | Still `unioffice`, now fully commercial-gated (confirmed both 2026-07-20 and re-confirmed 2026-07-21) | Ongoing since project's shift to AGPLv3/commercial licensing (predates this research) | No permissive Go PPTX-generation library exists; hand-rolling is not a stopgap, it is the only compliant option |
| Marp's experimental `--pptx-editable` (LibreOffice round-trip conversion) | N/A for Eden Press | Not adopted — Eden Press's approach avoids any LibreOffice runtime dependency in production, using it only as a CI verification tool | Confirms Eden Press's native-OOXML approach is a genuine upgrade over the closest upstream comparable, not mere parity |

**Deprecated/outdated:** Nothing OOXML-spec-side has changed; ECMA-376/ISO-29500 has been stable for PresentationML's core shape/text/EMU model for over a decade. The only "moving part" in this domain is the Go-ecosystem licensing landscape around `unioffice`, re-checked and unchanged this session.

## Open Questions

1. **`chase/model.Section` has no field for paragraph/list body content — how should Objective 6 be scoped?**
   - What we know: `chase/model/document.go` (read directly, current merged state from Objective 2/MODEL-01) defines `Section{ID, Attrs, Notes}` with an explicit doc comment stating body/Blocks content was deliberately excluded from this schema. `Document.Outline` supplies heading text (grouped by `SectionID`, flat, not nested). `press.Output.Model` is exactly this `Document` — there is no other channel to reach paragraph/list content without re-parsing HTML (which the objective explicitly forbids: "NOT rendered HTML").
   - What's unclear: Whether the planner/roadmap intends Objective 6 to (a) ship an honestly-reduced v1 that maps only Outline headings + Notes into shapes (this literally satisfies EXP-03's stated requirement — "editable text boxes... not a screenshot per slide" — without needing full paragraph/list fidelity), or (b) treat this as blocking on an undeclared new dependency (a small `chase/model` schema addition, e.g. `Section.Blocks`) that ROADMAP.md does not currently list Objective 6 as depending on (it currently lists only "Depends on: Objective 3").
   - Recommendation: Resolve explicitly at planning time. Recommended default: ship (a) — re-scope Objective 6's TRDs to "Outline headings + speaker notes," since that is both what EXP-03's literal text requires and what the current, already-merged docmodel can actually supply — and file the paragraph/list-body gap as a candidate for a future, separately-scoped docmodel objective rather than silently expanding Objective 6's dependency graph.

2. **Are there ANY currently-populated, well-known keys in `Section.Attrs` usable for slide-level styling (e.g. background color, class)?**
   - What we know: `Section.Attrs` is `map[string]string`, described as "directive-derived attribute values (data-*/style/class/lang/...)."
   - What's unclear: Whether `chase/directive` in practice populates specific, predictable keys (e.g. a background-color directive) that a PPTX writer could opportunistically consume for slide background/styling, or whether Attrs in practice is too free-form/unpredictable to build a mapping contract against without its own dedicated investigation of `chase/directive`.
   - Recommendation: Treat as out-of-scope/opportunistic for Objective 6's v1; if the planner wants slide-level styling fidelity, scope a small dedicated look at `chase/directive`'s actual directive set as a follow-up task rather than guessing here.

## Sources

### Primary (HIGH confidence)
- Microsoft Learn — Open XML SDK / ISO-IEC 29500 API reference, "Structure of a PresentationML document" and DrawingML `CT_GroupTransform2D` (`chOff`/`chExt`) reference pages — fetched directly this session, confirms zip part layout, relationship-type URIs, and group-transform coordinate-space semantics.
- `chase/model/document.go` (this repository) — read directly in full; ground truth for what `chase/model.Document`/`Section`/`OutlineEntry` actually contain (as opposed to what ARCHITECTURE.md's older prose description implies).
- `chase/model/build.go` (this repository) — read directly, confirms `Build()` populates only `Section.ID/Attrs/Notes` and `Document.Outline` from the AST walk; no Blocks/body-content construction path exists.
- `.planning/REQUIREMENTS.md` — direct read, confirms EXP-03's exact requirement text and MODEL-01..04 completion status.

### Secondary (MEDIUM confidence)
- officeopenxml.com (`drwGrp-chOffChExt.php`, `prSlide-shapeTree.php`, `anatomyofOOXML-pptx.php`) — content obtained via WebSearch result summaries this session; direct `WebFetch` to this domain failed both times (TLS handshake error `TLSV1_ALERT_INTERNAL_ERROR`). Cross-verified against the HIGH-confidence Microsoft Learn sources above where overlapping; no contradictions found.
- python-pptx documentation (packaging/minimal-part-set description) — WebSearch-verified, used as a cross-check on the minimal viable OPC part graph and on `fmtScheme`'s 3-entry cardinality requirement.
- WebSearch aggregation of Go `archive/zip` determinism practices (fixed `Modified` timestamp, `Store` vs `Deflate` cross-version stability, deterministic file-add ordering) — no single canonical stdlib doc page states this as a unified recipe; synthesized from multiple community sources plus `archive/zip`'s own godoc for `FileHeader`.
- WebSearch re-confirmation (2026-07-21) of Go PPTX-library licensing landscape — same result as `.planning/research/STACK.md`'s 2026-07-20 finding (`unioffice` + AGPLv3/commercial forks only, no permissive alternative).

### Tertiary (LOW confidence)
- None used as load-bearing for a specific factual claim in this document — all EMU constants, slide-size values, and clrMap/fmtScheme cardinality requirements were cross-verified against at least two independent sources before being stated as fact.

## Metadata

**Confidence breakdown:**
- Standard stack (stdlib-only, no third-party dep, licensing rejection of unioffice): HIGH — directly re-verified this session, consistent with prior STACK.md/PITFALLS.md research.
- Architecture (zip part layout, shape XML, EMU math, chOff/chExt): HIGH for the ECMA-376/ISO-29500-sourced facts (Microsoft Learn direct-fetched); MEDIUM for the officeopenxml.com-attributed specifics (WebSearch-mediated due to a direct-fetch TLS failure, but cross-verified against the HIGH-confidence source with no contradictions).
- Docmodel gap (Section has no body-content field): HIGH — verified by direct file read of `chase/model/document.go` and `build.go`, not inferred.
- Pitfalls (clrMap 12-attrs, fmtScheme 3-entries, determinism): MEDIUM-HIGH — spec-derived cardinality/attribute-completeness requirements, cross-verified across 2+ sources; Go-determinism specifics MEDIUM since no single canonical stdlib doc states it as a unified recipe.
- CI verification strategy (LibreOffice headless + structural asserts): MEDIUM — this is an industry-common pattern (used by python-pptx's and similar projects' own test suites per WebSearch) rather than a single authoritative spec statement, but it is the practical, CI-feasible proxy given no scriptable PowerPoint validator exists.

**Research date:** 2026-07-21
**Valid until:** ECMA-376/ISO-29500 core facts (EMU, zip layout, shape XML): effectively indefinite — this format has been stable for over a decade, re-check only if evidence of a spec revision emerges. Go-ecosystem licensing claim (no permissive PPTX library) and the `chase/model` schema-gap finding: re-verify at next objective-planning touchpoint (~30 days), since both could change independently (a new library could appear; `chase/model` could be extended by a future objective).
