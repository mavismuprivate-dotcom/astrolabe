# Frontend Copy Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove implementation-facing explanatory copy from the redesigned frontend so only user-facing content remains.

**Architecture:** Keep the current page structure and interactions intact. Add a regression test that asserts known implementation-commentary phrases are absent from `web/index.html`, then minimally edit the page copy to replace or remove those phrases.

**Tech Stack:** Go standard library tests, static HTML, CSS, vanilla JavaScript.

---

### Task 1: Add a regression test for non-user-facing copy

**Files:**
- Modify: `cmd/server/web_test.go`
- Test: `cmd/server/web_test.go`

**Step 1: Write the failing test**

Add a test that reads `web/index.html` and fails if it still contains these implementation-facing phrases:

- `当前表单字段、接口协议和交互行为保持不变`
- `当前页面保留的核心内容`
- `同一个挂载点渲染`
- `沿用当前渲染逻辑`
- `当前展示内容`
- `黑白高对比风格`
- `自动滚动到这里`
- `TODO 占位`

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/server -run TestIndexPageOmitsImplementationCommentary -v`

Expected: FAIL because the current page still contains those phrases.

### Task 2: Remove implementation-commentary copy

**Files:**
- Modify: `web/index.html`
- Test: `cmd/server/web_test.go`

**Step 1: Replace or delete the flagged copy**

Update only user-visible text. Do not change DOM IDs, form fields, or JavaScript behavior.

**Step 2: Run the focused test to verify it passes**

Run: `go test ./cmd/server -run TestIndexPageOmitsImplementationCommentary -v`

Expected: PASS

### Task 3: Full verification

**Files:**
- Verify: `cmd/server/web_test.go`
- Verify: `web/index.html`

**Step 1: Run the full Go test suite**

Run: `go test ./...`

Expected: PASS

**Step 2: Confirm key page hooks still exist**

Check that IDs used by the frontend logic remain in `web/index.html`.
