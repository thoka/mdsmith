package singleh1

import (
	"testing"

	goldast "github.com/jeduden/mdsmith/pkg/goldmark/ast"
	"github.com/jeduden/mdsmith/pkg/goldmark/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFix_H1InContainer_DemotesATX pins that the fix reads an ATX heading
// after a block quote or list marker as ATX: it adds a `#` to the heading
// and leaves the next line alone, instead of taking the heading for setext
// and rewriting `=` on the following line.
func TestFix_H1InContainer_DemotesATX(t *testing.T) {
	cases := map[string]string{
		"# First\n\n> # Second\n> a = b\n":   "# First\n\n> ## Second\n> a = b\n",
		"# First\n\n- # Second\n  a = b\n":   "# First\n\n- ## Second\n  a = b\n",
		"# First\n\n> - # Second\n>   ===\n": "# First\n\n> - ## Second\n>   ===\n",
		"# First\n\n> Second\n> ======\n":    "# First\n\n> Second\n> ------\n",
	}
	for src, want := range cases {
		got := (&Rule{}).Fix(newFile(t, src))
		assert.Equal(t, want, string(got), "source %q", src)
	}
}

// TestFix_MultiLineSetextH1_RewritesUnderline pins that the fix rewrites
// the underline after the last content line of a setext heading, not the
// line after its first content line.
func TestFix_MultiLineSetextH1_RewritesUnderline(t *testing.T) {
	src := "# First\n\nSecond\na = b\n=====\n"
	got := (&Rule{}).Fix(newFile(t, src))
	assert.Equal(t, "# First\n\nSecond\na = b\n-----\n", string(got))
}

// TestBuildDemoteReplacement_ATX_NoLines_WithTextChild covers the
// fallback for an ATX heading without Lines: the '#' goes after the up
// to 3 leading spaces of the line that holds the heading text.
func TestBuildDemoteReplacement_ATX_NoLines_WithTextChild(t *testing.T) {
	source := []byte("  # Title\n")
	h := goldast.NewHeading(1)
	child := goldast.NewText()
	child.Segment = text.NewSegment(4, 9)
	h.AppendChild(h, child)
	got, ok := buildDemoteReplacement(h, source)
	require.True(t, ok)
	assert.Equal(t, rep{start: 2, end: 2, newText: "#"}, got)
}
