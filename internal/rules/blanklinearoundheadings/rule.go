package blanklinearoundheadings

import (
	"bytes"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/astutil"
	"github.com/jeduden/mdsmith/pkg/goldmark/ast"
	"github.com/jeduden/mdsmith/pkg/markdown/flavor"
)

func init() {
	rule.Register(&Rule{})
}

// newlineSep is the bytes.Join separator; a package-level var avoids
// a heap allocation for []byte("\n") on every Fix call.
var newlineSep = []byte("\n")

// Rule checks that headings have blank lines before and after them.
type Rule struct{}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS013" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "blank-line-around-headings" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "heading" }

// Check implements rule.Rule. The per-heading logic depends only on the
// heading's line span and the blank lines around it — never on the inline
// tree — so the rule is a rule.BlockChecker: on a parsed File it folds
// into the engine's shared AST walk (rule.WalkNodes), and on a parse-
// skipped File (f.AST nil) it reads the Layer 0 block scan instead
// (rule.WalkBlocks). Both resolve the same heading line span, so the
// diagnostics are identical.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	if f != nil && f.AST == nil {
		return rule.WalkBlocks(r, f)
	}
	return rule.WalkNodes(r, f)
}

// CheckNode implements rule.NodeChecker. Code-line lookup runs lazily
// via lint.CollectCodeBlockLines (cached on f); the rule cannot
// precompute it once per Check because the engine multiplexes
// CheckNode calls, but the cache makes it a single walk per file
// across all callers.
func (r *Rule) CheckNode(n ast.Node, entering bool, f *lint.File) []lint.Diagnostic {
	if !entering {
		return nil
	}
	heading, ok := n.(*ast.Heading)
	if !ok {
		return nil
	}

	line := astutil.HeadingLine(heading, f)

	// Skip headings whose lines overlap with code block regions.
	codeLines := lint.CollectCodeBlockLines(f)
	if _, ok := codeLines[line]; ok {
		return nil
	}
	lastLine := headingLastLine(heading, f)
	return r.checkBlankLines(f, line, lastLine, headingQuoted(heading, f.Source))
}

// CheckBlock implements rule.BlockChecker. A heading span's 1-based
// Start is the first heading line (HeadingLine on the AST path) and End
// is its last line — the underline for a setext heading, the heading
// line itself for ATX — matching headingLastLine. The Layer 0 scanner
// never emits a heading span for a line inside a code block, so the
// AST path's code-line skip has no nil-AST counterpart to reproduce.
// A source with a block quote always takes the AST path
// (lint.SourceMayHaveBlockQuote), so a Layer 0 heading is never quoted.
func (r *Rule) CheckBlock(span lint.BlockSpan, f *lint.File) []lint.Diagnostic {
	return r.checkBlankLines(f, span.Start, span.End, false)
}

// blockKinds is the static block-kind interest CheckBlock declares via
// rule.BlockChecker; package-level so BlockKinds returns it without
// allocating. ATX and setext headings both map from ast.KindHeading.
var blockKinds = []lint.BlockKind{lint.BlockATXHeading, lint.BlockSetextHeading}

// BlockKinds implements rule.BlockChecker: CheckBlock reacts to both
// heading shapes.
func (r *Rule) BlockKinds() []lint.BlockKind { return blockKinds }

var _ rule.BlockChecker = (*Rule)(nil)

// checkBlankLines is the shared per-heading check both the AST
// (CheckNode) and Layer 0 (CheckBlock) paths drive: line is the 1-based
// first heading line, lastLine its 1-based last line. quoted reports
// that the heading sits in a block quote, where a line of only `>`
// markers is a blank line.
func (r *Rule) checkBlankLines(f *lint.File, line, lastLine int, quoted bool) []lint.Diagnostic {
	var diags []lint.Diagnostic

	// Check blank line before (not needed for line 1)
	if line > 1 {
		prevLineIdx := line - 2 // 0-based index
		if prevLineIdx >= 0 && prevLineIdx < len(f.Lines) {
			if isNonBlankLine(f.Lines, prevLineIdx, quoted) {
				diags = append(diags, lint.Diagnostic{
					File:     f.Path,
					Line:     line,
					Column:   1,
					RuleID:   r.ID(),
					RuleName: r.Name(),
					Severity: lint.Warning,
					Message:  "heading should have a blank line before",
				})
			}
		}
	}

	// Check blank line after (not needed for last line)
	nextLineIdx := lastLine // 0-based index of line after heading
	if nextLineIdx < len(f.Lines) {
		if isNonBlankLine(f.Lines, nextLineIdx, quoted) {
			diags = append(diags, lint.Diagnostic{
				File:     f.Path,
				Line:     line,
				Column:   1,
				RuleID:   r.ID(),
				RuleName: r.Name(),
				Severity: lint.Warning,
				Message:  "heading should have a blank line after",
			})
		}
	}
	return diags
}

// Fix implements rule.FixableRule.
func (r *Rule) Fix(f *lint.File) []byte {
	insertBefore, insertAfter := collectHeadingBlankLineInsertions(f)

	if len(insertBefore) == 0 && len(insertAfter) == 0 {
		return f.Source
	}

	// Pre-size to avoid growth allocations; worst case each line gets a
	// blank before and after. Work directly with []byte to avoid the
	// O(n) string([]byte) conversion that the previous string-join path
	// paid for every line in the document.
	result := make([][]byte, 0, len(f.Lines)+len(insertBefore)+len(insertAfter))
	for i, line := range f.Lines {
		lineNum := i + 1
		if blank, ok := insertBefore[lineNum]; ok {
			// Avoid inserting a blank line if one was just inserted
			// after the previous line (prevents double blank lines).
			if _, ok2 := insertAfter[lineNum-1]; !ok2 {
				result = append(result, blank)
			}
		}
		result = append(result, line)
		if blank, ok := insertAfter[lineNum]; ok {
			result = append(result, blank)
		}
	}

	return bytes.Join(result, newlineSep)
}

// collectHeadingBlankLineInsertions walks the AST and returns maps from
// 1-based line numbers that need a blank line inserted before or after
// them to the blank line to insert: nil at the top level, or the
// heading's block quote markers (blankLineFor) so the quote is not
// split. Insertion decisions are made directly inside the walk to avoid
// an intermediate slice and its growth allocations.
func collectHeadingBlankLineInsertions(f *lint.File) (insertBefore, insertAfter map[int][]byte) {
	insertBefore = make(map[int][]byte)
	insertAfter = make(map[int][]byte)
	codeLines := lint.CollectCodeBlockLines(f)
	lines := f.Lines

	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		line := astutil.HeadingLine(heading, f)
		if _, ok := codeLines[line]; ok {
			return ast.WalkContinue, nil
		}
		lastLine := headingLastLine(heading, f)
		quoted := headingQuoted(heading, f.Source)
		if line > 1 && isNonBlankLine(lines, line-2, quoted) {
			insertBefore[line] = blankLineFor(heading, f.Source)
		}
		if isNonBlankLine(lines, lastLine, quoted) {
			insertAfter[lastLine] = blankLineFor(heading, f.Source)
		}
		return ast.WalkContinue, nil
	})

	return insertBefore, insertAfter
}

// isNonBlankLine returns true if the 0-based index is within bounds and the
// line is non-blank. When quoted, the line's block quote markers are
// stripped first, so a line of only `>` markers is blank.
func isNonBlankLine(lines [][]byte, idx int, quoted bool) bool {
	if idx < 0 || idx >= len(lines) {
		return false
	}
	line := lines[idx]
	if quoted {
		line = flavor.StripBlockquoteMarkers(line)
	}
	return len(bytes.TrimSpace(line)) != 0
}

// headingPrefix returns the container prefix of the heading's first
// line (block quote markers, list markers, indentation), or nil when it
// is unknown.
func headingPrefix(heading *ast.Heading, source []byte) []byte {
	end, ok := astutil.HeadingPrefixEnd(heading, source)
	if !ok {
		return nil
	}
	start := end
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	return source[start:end]
}

// headingQuoted reports whether the heading's line carries a block
// quote marker in its container prefix.
func headingQuoted(heading *ast.Heading, source []byte) bool {
	return bytes.IndexByte(headingPrefix(heading, source), '>') >= 0
}

// blankLineFor returns the blank line to insert next to heading: its
// container prefix with list markers turned into spaces and trailing
// whitespace dropped. That keeps the block quote markers, so the new
// line stays inside the quote, and is nil outside a quote.
func blankLineFor(heading *ast.Heading, source []byte) []byte {
	if !headingQuoted(heading, source) {
		return nil
	}
	return bytes.TrimRight(astutil.ContinuationPrefix(headingPrefix(heading, source)), " \t")
}

func headingLastLine(heading *ast.Heading, f *lint.File) int {
	lines := heading.Lines()
	if lines.Len() > 0 {
		// Setext headings: the underline is on the line after the text
		lastSeg := lines.At(lines.Len() - 1)
		textLine := f.LineOfOffset(lastSeg.Start)
		// Check if next line is an underline (setext)
		if isSetextHeading(heading, f.Source) {
			return textLine + 1
		}
		return textLine
	}
	// ATX heading is a single line
	return astutil.HeadingLine(heading, f)
}

func isSetextHeading(heading *ast.Heading, source []byte) bool {
	lines := heading.Lines()
	if lines.Len() == 0 {
		return false
	}
	return !astutil.HeadingIsATX(heading, source)
}

var _ rule.FixableRule = (*Rule)(nil)
var _ rule.NodeChecker = (*Rule)(nil)

// FixTitle implements rule.QuickFixTitler.
func (r *Rule) FixTitle() string { return "Add blank lines around heading" }

// enteringKinds is the static node-kind interest CheckNode declares
// via rule.KindScopedChecker; package-level so EnteringKinds returns
// it without allocating.
var enteringKinds = []ast.NodeKind{ast.KindHeading}

// EnteringKinds implements rule.KindScopedChecker: CheckNode only
// reacts to these node kinds, entering visits only.
func (r *Rule) EnteringKinds() []ast.NodeKind { return enteringKinds }

var _ rule.KindScopedChecker = (*Rule)(nil)
