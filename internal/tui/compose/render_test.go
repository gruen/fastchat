package compose

import (
	"regexp"
	"strings"
	"testing"

	"github.com/mg/ai-tui/internal/db"
)

// ansiRe matches SGR (color/style) ANSI escape sequences emitted by glamour.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape sequences so assertions can reason about the
// visible text rather than glamour's exact byte output (which drifts across
// versions). Used by tests that need to check wording, not styling.
func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// lineCount returns the number of lines in s (after stripping ANSI), which is
// used to prove that narrower rendering wraps content onto more lines.
func lineCount(s string) int { return len(strings.Split(s, "\n")) }

// TestRenderMarkdown_StylesHeadingAndCodeBlock proves the acceptance criterion:
// a reply containing "# Heading" and a fenced code block renders with a styled
// heading and a code block rather than literal markdown syntax. Assertions are
// robust against glamour version drift: we check that output differs from the
// raw input, contains ANSI styling, still contains the heading word, consumes
// the code fence, and preserves the code content.
func TestRenderMarkdown_StylesHeadingAndCodeBlock(t *testing.T) {
	raw := "# Heading\n\nSome intro text.\n\n" +
		"```go\nfmt.Println(\"hi\")\n```\n"
	rendered := renderMarkdown(raw, 80)

	if rendered == raw {
		t.Fatalf("rendered output is identical to raw input; markdown was not rendered")
	}
	if !strings.Contains(rendered, "Heading") {
		t.Errorf("rendered output should still contain the heading word; got: %q", rendered)
	}
	if !strings.Contains(rendered, "\x1b[") {
		t.Errorf("rendered output should contain ANSI escape sequences (styling); got: %q", rendered)
	}
	// The fenced code markers must be consumed by the renderer (not literal).
	if strings.Contains(rendered, "```go") {
		t.Errorf("rendered output should not contain a literal ```go fence; got: %q", rendered)
	}
	// The code content itself must still be present (inside a styled block).
	if !strings.Contains(stripANSI(rendered), "Println") {
		t.Errorf("rendered output should preserve code content (Println); got: %q", rendered)
	}
}

// TestRenderMarkdown_NarrowWidthReflows proves that changing the width changes
// the rendered output and that narrower widths wrap content onto more lines.
func TestRenderMarkdown_NarrowWidthReflows(t *testing.T) {
	// A long paragraph that must wrap at both widths, plus a heading.
	raw := "# Heading\n\n" + strings.Repeat("lorem ipsum dolor sit amet ", 8) + "\n"

	out80 := renderMarkdown(raw, 80)
	out40 := renderMarkdown(raw, 40)

	if out80 == out40 {
		t.Fatalf("rendered output at width 80 and 40 are identical; width had no effect")
	}
	if lineCount(stripANSI(out40)) <= lineCount(stripANSI(out80)) {
		t.Errorf("narrower width (40) should wrap to more lines than wide (80): "+
			"lines40=%d lines80=%d", lineCount(stripANSI(out40)), lineCount(stripANSI(out80)))
	}
	// Both renderings must still contain the heading word.
	if !strings.Contains(stripANSI(out80), "Heading") || !strings.Contains(stripANSI(out40), "Heading") {
		t.Errorf("heading word should survive at both widths")
	}
}

// TestRenderMarkdown_ZeroWidthFallsBackToDefault proves a non-positive width
// does not panic and falls back to the default width instead of crashing.
func TestRenderMarkdown_ZeroWidthFallsBackToDefault(t *testing.T) {
	rendered := renderMarkdown("# Heading", 0)
	if !strings.Contains(rendered, "Heading") {
		t.Errorf("zero width should fall back to a default width and still render; got: %q", rendered)
	}
}

// TestUserMessagePlain_AssistantMessageRendered proves the scoping requirement:
// USER messages are kept as plain text (literal markdown markers preserved)
// while ASSISTANT messages are rendered through glamour (markers consumed).
func TestUserMessagePlain_AssistantMessageRendered(t *testing.T) {
	m := New(nil, nil)
	session := &db.Session{ID: "scope-1"}
	msgs := []db.Message{
		{SessionID: "scope-1", Role: "user", Content: "# Foo"},
		{SessionID: "scope-1", Role: "assistant", Content: "# Bar"},
	}
	m.LoadSession(session, msgs)

	view := m.View()
	plain := stripANSI(view)

	// User message is plain text: the literal "# Foo" marker survives.
	if !strings.Contains(plain, "# Foo") {
		t.Errorf("user message should remain plain text with literal '# Foo'; got: %q", plain)
	}
	// Assistant message is rendered: the word is present but the '#' marker is
	// consumed by glamour, so the literal "# Bar" must NOT appear.
	if !strings.Contains(plain, "Bar") {
		t.Errorf("assistant message should still contain the heading word 'Bar'; got: %q", plain)
	}
	if strings.Contains(plain, "# Bar") {
		t.Errorf("assistant message should be rendered (no literal '# Bar'); got: %q", plain)
	}
}

// TestStreamingBufferRenderedAsMarkdown proves the in-flight streaming buffer
// is rendered through glamour as it grows (the '#' marker is consumed while
// streaming), satisfying the streaming-render requirement.
func TestStreamingBufferRenderedAsMarkdown(t *testing.T) {
	m := New(nil, nil)
	m.streaming = true

	m, _ = m.Update(StreamChunkMsg{Content: "# Heading\n", Done: false})

	view := m.View()
	plain := stripANSI(view)

	if !strings.Contains(plain, "Heading") {
		t.Errorf("streaming buffer should contain the heading word; got: %q", plain)
	}
	if strings.Contains(plain, "# Heading") {
		t.Errorf("streaming buffer should be rendered (no literal '# Heading'); got: %q", plain)
	}
	// The raw (un-stripped) view must contain ANSI from the rendered heading,
	// proving the streaming buffer is styled, not passed through raw.
	if !strings.Contains(view, "\x1b[") {
		t.Errorf("streaming buffer should be styled with ANSI escapes; got: %q", view)
	}
}

// TestResizeReflowsAssistantMessage proves that resizing the viewport re-renders
// existing assistant messages so the rendered markdown reflows to the new width.
//
// The proof is split into two independent claims to avoid a viewport-height
// confound: the viewport has a FIXED visible height, so the total line count of
// m.View() stays roughly constant regardless of how the markdown wraps.
//
//  1. The pure renderMarkdown helper wraps content onto more lines at a narrow
//     width than a wide width (no viewport height in play).
//  2. SetSize triggers a re-render so m.View() actually differs between widths
//     (string inequality is enough; total View() line counts are NOT compared).
func TestResizeReflowsAssistantMessage(t *testing.T) {
	m := New(nil, nil)
	session := &db.Session{ID: "reflow-1"}
	long := strings.Repeat("lorem ipsum dolor sit amet ", 12)
	content := "# Title\n\n" + long
	msgs := []db.Message{
		{SessionID: "reflow-1", Role: "assistant", Content: content},
	}
	m.LoadSession(session, msgs)

	// (1) The pure rendering helper must wrap content onto more lines at a
	// narrow width than a wide width. This isolates the reflow claim from the
	// viewport's fixed visible height, which would otherwise mask differences in
	// total View() line count.
	out40 := renderMarkdown(content, 40)
	out80 := renderMarkdown(content, 80)
	if lineCount(stripANSI(out40)) <= lineCount(stripANSI(out80)) {
		t.Fatalf("renderMarkdown should wrap to more lines at width 40 than 80: "+
			"lines40=%d lines80=%d", lineCount(stripANSI(out40)), lineCount(stripANSI(out80)))
	}

	// (2) SetSize must trigger a re-render so the rendered markdown reflows to
	// the new viewport width. The viewport has a fixed visible height, so we
	// only assert the views DIFFER (string inequality) rather than comparing
	// total View() line counts, which stay roughly constant.
	m.SetSize(80, 60)
	viewWide := m.View()

	m.SetSize(40, 60)
	viewNarrow := m.View()

	if viewWide == viewNarrow {
		t.Fatalf("resize should reflow rendered output; wide and narrow views are identical")
	}
	// The heading word must survive both widths.
	if !strings.Contains(stripANSI(viewWide), "Title") || !strings.Contains(stripANSI(viewNarrow), "Title") {
		t.Errorf("heading word should survive resize reflow")
	}
}

// TestRendererCachedAndRecreatedOnWidthChange proves the renderer is cached on
// the Model and recreated only when the viewport width changes (the cache is
// invalidated and rebuilt, and output reflows to the new width).
func TestRendererCachedAndRecreatedOnWidthChange(t *testing.T) {
	m := New(nil, nil)

	m.viewport.Width = 80
	out80 := m.renderAssistant("# Heading")
	if m.renderer == nil {
		t.Fatal("renderer should be created on first render")
	}
	if m.renderW != 80 {
		t.Errorf("renderW should track the width used (80); got %d", m.renderW)
	}

	// A second render at the same width must reuse the cached renderer (no
	// recreation) and produce identical output.
	cached := m.renderer
	out80Again := m.renderAssistant("# Heading")
	if m.renderer != cached {
		t.Error("renderer should be reused when width is unchanged")
	}
	if out80 != out80Again {
		t.Error("re-rendering at the same width should produce identical output")
	}

	// Changing the width must invalidate the cache and recreate the renderer,
	// and the output must reflow to the new width.
	m.viewport.Width = 40
	out40 := m.renderAssistant("# Heading")
	if m.renderW != 40 {
		t.Errorf("renderW should update to new width (40); got %d", m.renderW)
	}
	if out80 == out40 {
		t.Error("rendering at a new width should reflow the output")
	}
}
