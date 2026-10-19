package astutil

import (
	"github.com/jeduden/mdsmith/pkg/goldmark/ast"
)

// HeadingIsATX reports whether heading is an ATX heading. goldmark starts
// an ATX heading's first content segment just after the opening `#` run
// and its spaces, so the byte before those spaces is a `#`. A setext
// heading's content starts right after its container prefix (block
// quote markers, a list marker, indentation), never after a `#`.
// Reading the marker back from the segment, not forward from the line
// start, keeps a heading inside a block quote or list item from looking
// setext. A heading with no Lines (an empty ATX heading such as `##`)
// is ATX: goldmark always records a setext heading's content line.
func HeadingIsATX(heading *ast.Heading, source []byte) bool {
	lines := heading.Lines()
	if lines.Len() == 0 {
		return true
	}
	i := skipSpacesBack(source, lines.At(0).Start)
	return i > 0 && source[i-1] == '#'
}

// HeadingPrefixEnd returns the byte offset where the container prefix of
// heading's first source line ends: the first `#` of an ATX heading or
// the first content byte of a setext heading. The bytes from the line
// start to that offset are the line's container prefix — block quote
// markers, list markers, and indentation. ok is false when heading has
// no Lines, so the offset is unknown.
func HeadingPrefixEnd(heading *ast.Heading, source []byte) (end int, ok bool) {
	lines := heading.Lines()
	if lines.Len() == 0 {
		return 0, false
	}
	start := lines.At(0).Start
	if !HeadingIsATX(heading, source) {
		return start, true
	}
	i := skipSpacesBack(source, start)
	for i > 0 && source[i-1] == '#' {
		i--
	}
	return i, true
}

// ContinuationPrefix maps a line's container prefix to the prefix that
// continues the same containers on a following line: block quote markers
// and whitespace stay, and each other byte (a list marker such as `-`
// or `1.`) becomes a space, so the next line sits in the same list item
// instead of opening a new one. It returns nil for an empty prefix.
func ContinuationPrefix(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	out := make([]byte, len(prefix))
	for i, c := range prefix {
		switch c {
		case '>', ' ', '\t':
			out[i] = c
		default:
			out[i] = ' '
		}
	}
	return out
}

// skipSpacesBack returns the offset before the run of spaces and tabs
// that ends at i on the same line.
func skipSpacesBack(source []byte, i int) int {
	for i > 0 && (source[i-1] == ' ' || source[i-1] == '\t') {
		i--
	}
	return i
}
