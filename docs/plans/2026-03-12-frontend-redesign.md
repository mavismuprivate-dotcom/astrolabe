# Astrolabe Frontend Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rebuild the single-page frontend into a video-backed landing page while preserving the existing astrolabe interactions and rendering hooks.

**Architecture:** Keep the app as a single static HTML file served by the existing Go server. Restructure the DOM into long-scroll sections, replace the CSS system, and preserve the existing JavaScript behavior by keeping all functional `id` hooks stable. Add one Go regression test that checks the landing-page skeleton so the refactor has at least one automated guardrail.

**Tech Stack:** Go standard library tests, static HTML, CSS, and vanilla JavaScript.

---

### Task 1: Add a regression test for the redesigned page shell

**Files:**
- Create: `cmd/server/web_test.go`
- Test: `cmd/server/web_test.go`

**Step 1: Write the failing test**

Add a test that reads `web/index.html` and expects the new layout anchors and background video markers to exist:

- `id="bg-video"`
- `id="hero"`
- `id="generator"`
- `id="chart-section"`
- `id="reading-section"`
- `id="todo-section"`

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/server -run TestIndexPageIncludesRedesignStructure -v`

Expected: FAIL because the current page does not yet contain the new IDs.

**Step 3: Keep the failing test in place**

Do not weaken the assertions to match the old page.

### Task 2: Rebuild the page structure and styling

**Files:**
- Modify: `web/index.html`
- Test: `cmd/server/web_test.go`

**Step 1: Replace the page shell**

Rebuild the HTML into:

- fixed navbar
- hero section
- generator section
- chart section
- reading section
- todo section

Keep the current functional nodes and IDs required by JavaScript:

- `#chart-form`
- `#error`
- `#meta`
- `#chart-wrap`
- `#evidence-list`
- all `#card-*` elements
- all `#metric-*` elements

**Step 2: Replace the CSS system**

Implement:

- `General Sans` font import and fallback stack
- black base background
- fullscreen looping background video with a 50% black overlay
- pill buttons and nav styling
- responsive section spacing
- stronger reading hierarchy
- mobile nav collapse behavior

**Step 3: Keep the JS behavior intact**

Only make the minimal JS changes needed to support the new layout, such as:

- anchor/button helpers if needed
- placeholder TODO interaction if needed
- any chart sizing adjustments required by the new section widths

**Step 4: Run the focused test to verify it passes**

Run: `go test ./cmd/server -run TestIndexPageIncludesRedesignStructure -v`

Expected: PASS

### Task 3: Run full verification

**Files:**
- Verify: `cmd/server/web_test.go`
- Verify: `web/index.html`

**Step 1: Run the full Go test suite**

Run: `go test ./...`

Expected: PASS across the repository.

**Step 2: Review the HTML hooks**

Check that all existing rendering and action IDs still exist in `web/index.html`.

**Step 3: Summarize residual risk**

Call out that responsive behavior and final visual polish were verified by code inspection and structure checks, but not by a browser screenshot test.
