# Astrolabe Frontend Redesign Design

**Date:** 2026-03-12

## Goal

Refactor the single-page frontend into a long-scroll landing page that keeps the current astrolabe content and interactions, while replacing the visual system with a black, video-backed, high-contrast style inspired by the reference description.

## Constraints

- Keep the current page content as the source of truth. Do not invent new product copy beyond minimal navigation and section framing.
- Keep the current page as a single HTML file served from `web/index.html`.
- Preserve existing front-end behavior:
  - form submission to `/api/v1/chart/natal`
  - natal chart SVG rendering in `#chart-wrap`
  - report rendering into the existing `id` targets
  - JSON download
  - report copy
  - sample data fill
- Map navbar items to real in-page anchors and actions where possible.
- Leave non-existent destinations as explicit TODO placeholders.

## Approved Direction

### Layout

Convert the current tool-style split layout into a long single-page flow:

1. Fixed top navigation
2. Hero section
3. Input section
4. Chart section
5. Reading section
6. TODO section for future capabilities

Each section receives a stable anchor so the navigation can scroll to it.

### Visual System

- Background: pure black base with a fullscreen looping background video.
- Readability: a 50% black overlay above the video.
- Typography: `General Sans` as the primary face, with existing Chinese system fallbacks.
- Palette: white and low-opacity white for text, outlines, and cards.
- Surfaces: restrained glass-like cards with subtle borders, blur, and a top-edge light streak.
- Buttons: pill-shaped, layered construction matching the reference style.

### Navbar

- Left: brand wordmark and Chinese navigation items.
- Right: a primary pill CTA.
- Desktop: full nav visible.
- Mobile: nav items hidden, keep compact brand + CTA.

Navigation mapping:

- `开始生成` -> `#generator`
- `星盘展示` -> `#chart-section`
- `解读内容` -> `#reading-section`
- `更多功能` -> `#todo-section`
- Primary CTA -> `#generator`

### Hero

The hero keeps the reference composition but uses current product context:

- badge with early-access framing
- large gradient heading
- concise project description
- primary CTA that scrolls to the form section

The hero is purely presentational and does not replace existing functionality.

### Content Sections

#### Input section

- Keep all current fields and buttons.
- Group them inside a stronger editorial section rather than a utilitarian side panel.
- Maintain the current field names and form `id`.

#### Chart section

- Promote the natal chart into its own large feature section.
- Preserve `#chart-wrap` and chart rendering logic.

#### Reading section

- Recompose the summary, themes, evidence, quality metrics, and global notes into vertically stacked sections.
- Preserve all current `id` hooks so rendering logic remains intact.

#### TODO section

- Add a visually consistent placeholder section for future capabilities.
- Use existing current-context Chinese copy rather than reference-site marketing copy.

## Interaction Notes

- Navbar links use in-page anchor scrolling.
- CTA buttons scroll to the generator section.
- Existing form actions remain functionally unchanged.
- Preserve hover and focus affordances while aligning them with the new monochrome system.

## Error Handling and Empty States

- Keep the existing error and meta areas, but restyle them for better prominence.
- Show placeholder content in chart and reading sections before generation.
- Ensure the page remains readable if the video fails to load.

## Testing Strategy

- Add a regression test that asserts the redesigned page contains the key long-scroll structure:
  - video background element
  - hero anchor
  - generator anchor
  - chart anchor
  - reading anchor
  - todo anchor
- Verify the test fails before the HTML refactor and passes after it.
- Run the full Go test suite after the change.

## Files Expected To Change

- `web/index.html`
- `cmd/server/web_test.go`
- `docs/plans/2026-03-12-frontend-redesign-design.md`
- `docs/plans/2026-03-12-frontend-redesign.md`
