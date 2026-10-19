package astutil

import (
	"testing"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/pkg/goldmark/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseFirstHeading parses src and returns its first heading.
func parseFirstHeading(t *testing.T, src string) (*ast.Heading, []byte) {
	t.Helper()
	f, err := lint.NewFile("test.md", []byte(src))
	require.NoError(t, err)
	var h *ast.Heading
	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if hh, ok := n.(*ast.Heading); ok && entering && h == nil {
			h = hh
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	require.NotNil(t, h, "no heading in %q", src)
	return h, f.Source
}

func TestHeadingIsATX(t *testing.T) {
	cases := map[string]bool{
		"# Foo\n":            true,
		"   ## Foo\n":        true,
		"##\n":               true,
		"> ### Foo {#id}\n":  true,
		"> > # Foo\n":        true,
		"- # Foo\n":          true,
		"> - ## Foo\n":       true,
		"Foo\n===\n":         false,
		"> Foo\n> ---\n":     false,
		"- #5 Foo\n  ===\n":  false,
		"> \\# Foo\n> ===\n": false,
	}
	for src, want := range cases {
		h, source := parseFirstHeading(t, src)
		assert.Equal(t, want, HeadingIsATX(h, source), "source %q", src)
	}
}

func TestHeadingPrefixEnd(t *testing.T) {
	cases := map[string]string{
		"# Foo\n":          "",
		"  # Foo\n":        "  ",
		"> ### Foo\n":      "> ",
		"> - # Foo\n":      "> - ",
		"- > ## Foo\n":     "- > ",
		"Foo\n===\n":       "",
		"> > Foo\n> > -\n": "> > ",
	}
	for src, want := range cases {
		h, source := parseFirstHeading(t, src)
		end, ok := HeadingPrefixEnd(h, source)
		require.True(t, ok, "source %q", src)
		assert.Equal(t, want, src[:end], "source %q", src)
	}
}

func TestHeadingPrefixEnd_NoLines(t *testing.T) {
	h, source := parseFirstHeading(t, "##\n")
	_, ok := HeadingPrefixEnd(h, source)
	assert.False(t, ok)
}

func TestContinuationPrefix(t *testing.T) {
	cases := map[string]string{
		"":      "",
		"  ":    "  ",
		"> ":    "> ",
		"> - ":  ">   ",
		"- > ":  "  > ",
		"10. ":  "    ",
		">\t> ": ">\t> ",
	}
	for in, want := range cases {
		assert.Equal(t, want, string(ContinuationPrefix([]byte(in))), "prefix %q", in)
	}
	assert.Nil(t, ContinuationPrefix(nil))
}

func TestSkipSpacesBack(t *testing.T) {
	src := []byte("> #  \tx")
	assert.Equal(t, 3, skipSpacesBack(src, 6))
	assert.Equal(t, 0, skipSpacesBack([]byte("  x"), 2))
	assert.Equal(t, 1, skipSpacesBack(src, 1))
}
