package blanklinearoundheadings

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/pkg/goldmark/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheck_HeadingInQuote_QuoteBlankCounts pins that a quote line with
// only `>` markers is a blank line inside the quote, and that an ATX
// heading in a container is one line long (not read as setext).
func TestCheck_HeadingInQuote_QuoteBlankCounts(t *testing.T) {
	srcs := []string{
		"> # H\n>\n> text\n",
		"> text\n>\n> # H\n>\n> more\n",
		"> > # H\n> >\n> > text\n",
		"- > # H\n  >\n  > text\n",
	}
	for _, src := range srcs {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Empty(t, (&Rule{}).Check(f), "source %q", src)
	}
}

// TestCheck_HeadingInContainer_NoBlankAfter keeps the true finding: a
// heading inside a container that a paragraph follows directly.
func TestCheck_HeadingInContainer_NoBlankAfter(t *testing.T) {
	srcs := []string{
		"> # H\n> text\n",
		"- # H\n  text\n",
	}
	for _, src := range srcs {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		diags := (&Rule{}).Check(f)
		require.Len(t, diags, 1, "source %q", src)
		assert.Equal(t, 1, diags[0].Line)
		assert.Equal(t, "heading should have a blank line after", diags[0].Message)
	}
}

// reportSrc is the reported case: an ATX heading with an attribute in a
// block quote, directly followed by a quoted paragraph.
const (
	reportSrc  = "# Title\n\n> ### R {#id}\n> status: open\n>\n> Prose.\n"
	reportWant = "# Title\n\n> ### R {#id}\n>\n> status: open\n>\n> Prose.\n"
)

// TestFix_HeadingInContainer_InsertsPrefixedBlank pins that the fix
// inserts a blank line that keeps the quote markers of the heading, so
// the quote is not split and no line leaves it.
func TestFix_HeadingInContainer_InsertsPrefixedBlank(t *testing.T) {
	cases := map[string]string{
		"> # H\n> text\n":            "> # H\n>\n> text\n",
		"> > # H\n> > text\n":        "> > # H\n> >\n> > text\n",
		"> text\n> # H\n>\n> more\n": "> text\n>\n> # H\n>\n> more\n",
		"- > # H\n  > text\n":        "- > # H\n  >\n  > text\n",
		"> - # H\n>   text\n":        "> - # H\n>\n>   text\n",
		"- # H\n  text\n":            "- # H\n\n  text\n",
		reportSrc:                    reportWant,
	}
	for src, want := range cases {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Equal(t, want, string((&Rule{}).Fix(f)), "source %q", src)
	}
}

// firstHeadingOf parses src and returns its first heading node.
func firstHeadingOf(t *testing.T, src string) (*ast.Heading, []byte) {
	t.Helper()
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	var h *ast.Heading
	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if hh, ok := n.(*ast.Heading); ok && entering && h == nil {
			h = hh
		}
		return ast.WalkContinue, nil
	})
	require.NotNil(t, h)
	return h, f.Source
}

func TestHeadingPrefix(t *testing.T) {
	cases := map[string]string{
		"# H\n":       "",
		"> - # H\n":   "> - ",
		"- > ## H\n":  "- > ",
		"> H\n> ==\n": "> ",
	}
	for src, want := range cases {
		h, source := firstHeadingOf(t, src)
		assert.Equal(t, want, string(headingPrefix(h, source)), "source %q", src)
	}
	h, source := firstHeadingOf(t, "> ##\n")
	assert.Nil(t, headingPrefix(h, source), "an empty heading has no Lines")
}

func TestHeadingQuoted(t *testing.T) {
	cases := map[string]bool{
		"# H\n":     false,
		"- # H\n":   false,
		"> # H\n":   true,
		"- > # H\n": true,
	}
	for src, want := range cases {
		h, source := firstHeadingOf(t, src)
		assert.Equal(t, want, headingQuoted(h, source), "source %q", src)
	}
}

func TestBlankLineFor(t *testing.T) {
	cases := map[string]string{
		"# H\n":      "",
		"- # H\n":    "",
		"> # H\n":    ">",
		"> > # H\n":  "> >",
		"> - # H\n":  ">",
		"- > # H\n":  "  >",
		"> 1. # H\n": ">",
	}
	for src, want := range cases {
		h, source := firstHeadingOf(t, src)
		assert.Equal(t, want, string(blankLineFor(h, source)), "source %q", src)
	}
}

func TestIsNonBlankLine_Quoted(t *testing.T) {
	lines := [][]byte{[]byte(">"), []byte("> >  "), []byte("> text"), []byte("  >")}
	assert.False(t, isNonBlankLine(lines, 0, true))
	assert.False(t, isNonBlankLine(lines, 1, true))
	assert.True(t, isNonBlankLine(lines, 2, true))
	assert.False(t, isNonBlankLine(lines, 3, true))
	assert.True(t, isNonBlankLine(lines, 0, false), "outside a quote `>` is content")
}
