package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeduden/mdsmith/internal/config"
	"github.com/jeduden/mdsmith/internal/fix"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFix_HeadingInBlockQuote_KeepsQuote runs the default fixers on an
// ATX heading inside a block quote. MDS002 once read the heading as
// setext and its fix replaced two lines, so the heading left the quote
// and the next quote line was lost; MDS013 inserted a bare blank line
// that split the quote. Every source line must survive, and every line
// of the quote must stay in the quote.
func TestFix_HeadingInBlockQuote_KeepsQuote(t *testing.T) {
	src := "# Title\n\n" +
		"> ### Rebuild the writer {#JbWv0EAr}\n" +
		"> status: open\n" +
		">\n" +
		"> Some prose of the node.\n\n" +
		"{<JbWv0EAr}\n"

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(src), 0o644))

	result := (&fix.Fixer{Config: config.Defaults(), Rules: rule.All()}).Fix([]string{mdFile})
	require.Empty(t, result.Errors, "unexpected errors: %v", result.Errors)

	got, err := os.ReadFile(mdFile)
	require.NoError(t, err)
	content := string(got)

	for _, line := range strings.Split(strings.TrimSuffix(src, "\n"), "\n") {
		assert.Contains(t, content, line+"\n", "line %q was lost", line)
	}
	want := "# Title\n\n" +
		"> ### Rebuild the writer {#JbWv0EAr}\n" +
		">\n" +
		"> status: open\n" +
		">\n" +
		"> Some prose of the node.\n\n" +
		"{<JbWv0EAr}\n"
	assert.Equal(t, want, content)
}
