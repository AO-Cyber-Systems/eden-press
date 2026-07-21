---
work: feature
requirements: [EXP-03]
depends_on: [3]
---
# convert/pptx (native OOXML)
## Goal
Deliver editable-text-box PPTX export directly from chase/model (via press.Output.Model), Chrome-free, hand-rolled OOXML (archive/zip + encoding/xml). Sibling to Obj 5, not sequential.
## Decision gate (re-confirm at planning)
Hand-rolled OOXML confirmed; unioffice + forks rejected (AGPLv3 / commercial license-key). Re-confirm no new permissive Go PPTX lib emerged since research.
## Requirements
EXP-03 (see .planning/REQUIREMENTS.md)
---
*Created: 2026-07-21 (/devflow:build parallel workstreams)*
