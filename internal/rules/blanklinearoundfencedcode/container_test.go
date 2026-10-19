package blanklinearoundfencedcode

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/pkg/goldmark/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheck_FenceInBlockQuote_QuotedBlankCounts pins that a line of only
// `>` markers is a blank line inside a block quote, so a fence framed by
// such lines draws no diagnostic.
func TestCheck_FenceInBlockQuote_QuotedBlankCounts(t *testing.T) {
	for _, src := range []string{
		"> text\n>\n> ```\n> code\n> ```\n>\n> more\n",
		"- > text\n  >\n  > ```\n  > code\n  > ```\n  >\n  > more\n",
	} {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Empty(t, (&Rule{}).Check(f), "source %q", src)
	}
}

// TestFix_FenceInBlockQuote_InsertsPrefixedBlank pins that the fix
// inserts a blank line that keeps the quote markers of the fence, so the
// quote is not split into two quotes.
func TestFix_FenceInBlockQuote_InsertsPrefixedBlank(t *testing.T) {
	cases := map[string]string{
		"> text\n> ```\n> code\n> ```\n> more\n": "> text\n>\n> ```\n> code\n> ```\n>\n> more\n",
		"> > text\n> > ```\n> > ```\n":           "> > text\n> >\n> > ```\n> > ```\n",
		"- > text\n  > ```\n  > ```\n  > more\n": "- > text\n  >\n  > ```\n  > ```\n  >\n  > more\n",
		"text\n```\ncode\n```\nmore\n":           "text\n\n```\ncode\n```\n\nmore\n",
	}
	for src, want := range cases {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Equal(t, want, string((&Rule{}).Fix(f)), "source %q", src)
	}
}

// TestQuoteBlankLine_OutOfRange pins that a line number outside the file
// yields no blank line instead of an index panic.
func TestQuoteBlankLine_OutOfRange(t *testing.T) {
	f, err := lint.NewFile("test.md", []byte("> ```\n> ```\n"))
	require.NoError(t, err)
	fcb, ok := f.AST.FirstChild().FirstChild().(*ast.FencedCodeBlock)
	require.True(t, ok)
	assert.Equal(t, ">", string(quoteBlankLine(f, fcb, 1)))
	assert.Nil(t, quoteBlankLine(f, fcb, 0))
	assert.Nil(t, quoteBlankLine(f, fcb, len(f.Lines)+1))
}
