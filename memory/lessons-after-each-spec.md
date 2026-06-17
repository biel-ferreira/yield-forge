---
name: lessons-after-each-spec
description: After finishing each SPEC, produce a didactic HTML lesson in Portuguese recapping what was built
metadata:
  type: feedback
---

After completing each SPEC, the user wants a detailed, didactic explanation **in
Portuguese (PT-BR)** of everything that was built, delivered as a **self-contained
HTML file** (inline CSS, no external deps) so it's easy to open and read in a
browser — for their learning.

**Why:** they are learning Go + AI engineering by building this project and want to
consolidate knowledge spec-by-spec, not just have working code.

**How to apply:** when a SPEC is closed (after the docs phase), write an HTML lesson
under `docs/lessons/` named for the spec (e.g. `SPEC-001-aula.html`). Cover *what*
was built, *why* (the reasoning/trade-offs), and the *key concepts* learned (Go
idioms, patterns, tools). Keep it visual: sectioned with a table of contents, code
blocks, simple diagrams, and "Por quê?" callout boxes. Escape `<`/`>`/`&` inside
code blocks. Link [[yieldforge-project]].
