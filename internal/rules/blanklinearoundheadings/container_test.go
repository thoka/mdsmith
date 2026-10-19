package blanklinearoundheadings

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
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
		"# Title\n\n> ### R {#id}\n> status: open\n>\n> Prose.\n": "# Title\n\n> ### R {#id}\n>\n> status: open\n>\n> Prose.\n",
	}
	for src, want := range cases {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Equal(t, want, string((&Rule{}).Fix(f)), "source %q", src)
	}
}
