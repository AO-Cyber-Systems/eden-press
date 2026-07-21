---
work: feature
requirements: [EXP-01, EXP-02, EXP-04]
depends_on: [3]
---
# convert/pdf + convert/png (chromedp raster export)
## Goal
Deliver PDF and PNG/JPEG export via headless Chrome, isolated as the ONLY Chrome-touching code in the module, with CI-hardened determinism. The Obj-3 no-chromedp gate on press/chase/profiles must stay green.
## Requirements
EXP-01, EXP-02, EXP-04 (see .planning/REQUIREMENTS.md)
---
*Created: 2026-07-21 (/devflow:build parallel workstreams)*
