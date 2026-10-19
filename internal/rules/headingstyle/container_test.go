package headingstyle

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheck_ATXInContainer_NoViolation pins that an ATX heading inside a
// block quote or a list item is read as ATX: the container prefix (`>`,
// a list marker) before the `#` run must not make it look setext.
func TestCheck_ATXInContainer_NoViolation(t *testing.T) {
	srcs := []string{
		"# Title\n\n> ### Quoted {#id}\n> status: open\n>\n> Prose.\n",
		"# Title\n\n> > ## Nested\n",
		"# Title\n\n- # In a list item\n",
		"# Title\n\n> - ### List in quote\n",
		"# Title\n\n- > ## Quote in list\n",
	}
	for _, src := range srcs {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Empty(t, (&Rule{Style: "atx"}).Check(f), "source %q", src)
	}
}

// TestCheck_SetextInQuote_Violation keeps the true finding: a setext
// heading inside a block quote is still setext.
func TestCheck_SetextInQuote_Violation(t *testing.T) {
	f, err := lint.NewFile("test.md", []byte("# Title\n\n> Quoted\n> ======\n"))
	require.NoError(t, err)
	diags := (&Rule{Style: "atx"}).Check(f)
	require.Len(t, diags, 1)
	assert.Equal(t, 3, diags[0].Line)
}

// TestFix_ATXInContainer_Unchanged pins that the fix never rewrites a
// quoted or listed ATX heading, so no quote line is lost.
func TestFix_ATXInContainer_Unchanged(t *testing.T) {
	src := "# Title\n\n> ### Quoted {#id}\n> status: open\n>\n> Prose.\n"
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	assert.Equal(t, src, string((&Rule{Style: "atx"}).Fix(f)))
}

// TestFix_SetextToATX_KeepsContainerPrefix pins that converting a
// setext heading inside a container keeps the container prefix.
func TestFix_SetextToATX_KeepsContainerPrefix(t *testing.T) {
	cases := map[string]string{
		"> Foo\n> ===\n":            "> # Foo\n",
		"> > Foo\n> > ---\n":        "> > ## Foo\n",
		"- Foo\n  ---\n":            "- ## Foo\n",
		"> text\n>\n> Foo\n> ===\n": "> text\n>\n> # Foo\n",
	}
	for src, want := range cases {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Equal(t, want, string((&Rule{Style: "atx"}).Fix(f)), "source %q", src)
	}
}

// TestFix_ATXToSetext_PrefixesUnderline pins that converting an ATX
// heading inside a container keeps the prefix on the heading line and
// repeats the container prefix on the new underline line.
func TestFix_ATXToSetext_PrefixesUnderline(t *testing.T) {
	cases := map[string]string{
		"> # Foo\n":    "> Foo\n> ===\n",
		"> > ## Foo\n": "> > Foo\n> > ---\n",
		"- # Foo\n":    "- Foo\n  ===\n",
		"> - # Foo\n":  "> - Foo\n>   ===\n",
	}
	for src, want := range cases {
		f, err := lint.NewFile("test.md", []byte(src))
		require.NoError(t, err)
		assert.Equal(t, want, string((&Rule{Style: "setext"}).Fix(f)), "source %q", src)
	}
}
