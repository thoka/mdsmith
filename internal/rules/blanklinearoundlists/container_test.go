package blanklinearoundlists

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheck_ListInBlockQuote_QuotedBlankCounts pins that a line of only
// `>` markers is a blank line inside a block quote, so a list framed by
// such lines draws no diagnostic.
func TestCheck_ListInBlockQuote_QuotedBlankCounts(t *testing.T) {
	for _, src := range []string{
		"> text\n>\n> - a\n> - b\n>\n> # After\n",
		"- > text\n  >\n  > - a\n  >\n  > # After\n",
	} {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Empty(t, (&Rule{}).Check(f), "source %q", src)
	}
}

// TestFix_ListInBlockQuote_InsertsPrefixedBlank pins that the fix inserts
// a blank line that keeps the quote markers of the list, so the quote is
// not split into two quotes.
func TestFix_ListInBlockQuote_InsertsPrefixedBlank(t *testing.T) {
	cases := map[string]string{
		"> text\n> - a\n> # After\n":       "> text\n>\n> - a\n>\n> # After\n",
		"> > text\n> > - a\n":              "> > text\n> >\n> > - a\n",
		"- > text\n  > - a\n  > # After\n": "- > text\n  >\n  > - a\n  >\n  > # After\n",
		"text\n- a\n# After\n":             "text\n\n- a\n\n# After\n",
	}
	for src, want := range cases {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Equal(t, want, string((&Rule{}).Fix(f)), "source %q", src)
	}
}
