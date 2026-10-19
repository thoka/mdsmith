package blanklinearoundlists

import (
	"bytes"

	"github.com/jeduden/mdsmith/internal/lint"
	"github.com/jeduden/mdsmith/internal/rule"
	"github.com/jeduden/mdsmith/internal/rules/astutil"
	"github.com/jeduden/mdsmith/internal/rules/listscan"
	"github.com/jeduden/mdsmith/pkg/goldmark/ast"
	"github.com/jeduden/mdsmith/pkg/markdown/flavor"
)

func init() {
	rule.Register(&Rule{})
}

// Rule checks that lists have blank lines before and after them.
type Rule struct{}

// ID implements rule.Rule.
func (r *Rule) ID() string { return "MDS014" }

// Name implements rule.Rule.
func (r *Rule) Name() string { return "blank-line-around-lists" }

// Category implements rule.Rule.
func (r *Rule) Category() string { return "list" }

// Check implements rule.Rule. The per-list logic reads only a top-level
// list's first and last source lines and the blank lines around them —
// never the inline tree — so on a parsed File it folds into the engine's
// shared AST walk (rule.WalkNodes), and on a parse-skipped File (f.AST
// nil) it re-derives the same top-level list spans from f.Lines via
// listscan (checkLayer0). Both resolve the same span edges, so the
// diagnostics are identical.
func (r *Rule) Check(f *lint.File) []lint.Diagnostic {
	if f != nil && f.AST == nil {
		return r.checkLayer0(f)
	}
	return rule.WalkNodes(r, f)
}

// checkLayer0 is the nil-AST counterpart of CheckNode. listscan reports
// the same top-level lists goldmark would, with the same first and last
// source lines, so the blank-line verdict around each is byte-identical.
func (r *Rule) checkLayer0(f *lint.File) []lint.Diagnostic {
	lists := listscan.ParseLists(f.Lines)
	codeLines := lint.CollectCodeBlockLines(f)
	var diags []lint.Diagnostic
	for _, l := range lists {
		if !l.TopLevel {
			continue
		}
		diags = append(diags, r.verdict(f, codeLines, l.FirstLine, l.LastLine, false)...)
	}
	return diags
}

// LinesCapable implements rule.LinesChecker: the rule's Check serves the
// nil-AST (parse-skip) path itself by re-deriving list structure from
// f.Lines via listscan, so the engine routes it to Check on a skipped File
// instead of dropping it. Always true.
func (r *Rule) LinesCapable() bool { return true }

// CheckNode implements rule.NodeChecker.
func (r *Rule) CheckNode(n ast.Node, entering bool, f *lint.File) []lint.Diagnostic {
	if !entering {
		return nil
	}
	list, ok := n.(*ast.List)
	if !ok {
		return nil
	}
	if _, isListItem := list.Parent().(*ast.ListItem); isListItem {
		return nil
	}
	start := lineOfNode(f, list)
	quoted := quoteBlankLine(f, list, start) != nil
	return r.verdict(f, lint.CollectCodeBlockLines(f), start, lastLineOfNode(f, list), quoted)
}

// verdict is the shared per-list check both the AST (CheckNode) and Layer
// 0 (checkLayer0) paths drive: listStartLine and listEndLine are the
// top-level list's 1-based first and last source lines. A list whose edge
// sits inside a code block is skipped, matching the AST path. quoted
// reports that the list sits in a block quote, where a line of only `>`
// markers is a blank line. A source with a block quote always takes the
// AST path, so the Layer 0 path passes false.
func (r *Rule) verdict(
	f *lint.File, codeLines map[int]struct{}, listStartLine, listEndLine int, quoted bool,
) []lint.Diagnostic {
	if listStartLine <= 0 || listEndLine <= 0 {
		return nil
	}
	if lineInSet(codeLines, listStartLine) || lineInSet(codeLines, listEndLine) {
		return nil
	}
	var diags []lint.Diagnostic
	if d, ok := r.checkAdjacentBlank(f, listStartLine, -1, quoted, "list should be preceded by a blank line"); ok {
		diags = append(diags, d)
	}
	if d, ok := r.checkAdjacentBlank(f, listEndLine, +1, quoted, "list should be followed by a blank line"); ok {
		diags = append(diags, d)
	}
	return diags
}

var _ rule.NodeChecker = (*Rule)(nil)

// checkAdjacentBlank checks whether the line adjacent to targetLine (offset -1 for before,
// +1 for after) is non-blank and returns a diagnostic if so.
func (r *Rule) checkAdjacentBlank(
	f *lint.File, targetLine, direction int, quoted bool, msg string,
) (lint.Diagnostic, bool) {
	if !needsBlankAdjacent(f, targetLine, direction, quoted) {
		return lint.Diagnostic{}, false
	}
	return lint.Diagnostic{
		File:     f.Path,
		Line:     targetLine,
		Column:   1,
		RuleID:   r.ID(),
		RuleName: r.Name(),
		Severity: lint.Warning,
		Message:  msg,
	}, true
}

// isInlineNode returns true for inline AST nodes whose Lines() method panics.
func isInlineNode(n ast.Node) bool {
	switch n.(type) {
	case *ast.Text, *ast.String, *ast.CodeSpan, *ast.Emphasis,
		*ast.Link, *ast.Image, *ast.AutoLink, *ast.RawHTML:
		return true
	}
	return false
}

func lineOfNode(f *lint.File, n ast.Node) int {
	// Inline nodes do not have Lines(); use Segment for Text nodes.
	if t, ok := n.(*ast.Text); ok {
		return f.LineOfOffset(t.Segment.Start)
	}
	if isInlineNode(n) {
		// For other inline nodes, recurse into children.
		if n.HasChildren() {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				line := lineOfNode(f, c)
				if line > 0 {
					return line
				}
			}
		}
		return 0
	}
	if n.Lines().Len() > 0 {
		seg := n.Lines().At(0)
		return f.LineOfOffset(seg.Start)
	}
	// For container nodes, find the first child with lines.
	if n.HasChildren() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			line := lineOfNode(f, c)
			if line > 0 {
				return line
			}
		}
	}
	return 0
}

func lastLineOfNode(f *lint.File, n ast.Node) int {
	// Inline nodes do not have Lines(); use Segment for Text nodes.
	if t, ok := n.(*ast.Text); ok {
		return f.LineOfOffset(t.Segment.Stop - 1)
	}
	if isInlineNode(n) {
		if n.HasChildren() {
			for c := n.LastChild(); c != nil; c = c.PreviousSibling() {
				line := lastLineOfNode(f, c)
				if line > 0 {
					return line
				}
			}
		}
		return 0
	}
	if n.Lines().Len() > 0 {
		seg := n.Lines().At(n.Lines().Len() - 1)
		return f.LineOfOffset(seg.Start)
	}
	// For container nodes, find the last child with lines.
	if n.HasChildren() {
		for c := n.LastChild(); c != nil; c = c.PreviousSibling() {
			line := lastLineOfNode(f, c)
			if line > 0 {
				return line
			}
		}
	}
	return 0
}

// isBlank reports whether line is blank. When quoted, the line's block
// quote markers are stripped first, so a line of only `>` markers is
// blank.
func isBlank(line []byte, quoted bool) bool {
	if quoted {
		line = flavor.StripBlockquoteMarkers(line)
	}
	return len(bytes.TrimSpace(line)) == 0
}

// quoteBlankLine returns the blank line to insert next to list, whose
// first source line is the 1-based line: the list's block quote markers
// (astutil.QuoteBlankLine), or nil outside a block quote.
func quoteBlankLine(f *lint.File, list *ast.List, line int) []byte {
	if line < 1 || line > len(f.Lines) {
		return nil
	}
	return astutil.QuoteBlankLine(list, f.Lines[line-1])
}

// lineInSet reports whether the 1-based line is a member of set. It
// wraps the comma-ok membership test so callers can compose two
// checks in a single short-circuiting `||` expression — when the
// first line is already in a code block the second lookup is skipped.
func lineInSet(set map[int]struct{}, line int) bool {
	_, ok := set[line]
	return ok
}

// Fix implements rule.FixableRule.
func (r *Rule) Fix(f *lint.File) []byte {
	beforeSet, afterSet := r.collectBlankLineInsertions(f)

	if len(beforeSet) == 0 && len(afterSet) == 0 {
		return f.Source
	}

	// Upper bound: each line may need a blank inserted before and after it.
	resultLines := make([][]byte, 0, len(f.Lines)+len(beforeSet)+len(afterSet))
	for i, line := range f.Lines {
		lineNum := i + 1
		if blank, ok := beforeSet[lineNum]; ok {
			resultLines = append(resultLines, blank)
		}
		resultLines = append(resultLines, line)
		if blank, ok := afterSet[lineNum]; ok {
			resultLines = append(resultLines, blank)
		}
	}

	return bytes.Join(resultLines, newlineSep)
}

// collectBlankLineInsertions walks the AST and returns maps from 1-based
// line numbers that need a blank line inserted before or after them to
// the blank line to insert: nil at the top level, or the list's block
// quote markers (quoteBlankLine) so the quote is not split.
func (r *Rule) collectBlankLineInsertions(f *lint.File) (beforeSet, afterSet map[int][]byte) {
	beforeSet = make(map[int][]byte)
	afterSet = make(map[int][]byte)
	codeLines := lint.CollectCodeBlockLines(f)

	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		list, ok := n.(*ast.List)
		if !ok {
			return ast.WalkContinue, nil
		}

		if _, isListItem := list.Parent().(*ast.ListItem); isListItem {
			return ast.WalkContinue, nil
		}

		listStartLine := lineOfNode(f, list)
		listEndLine := lastLineOfNode(f, list)

		if lineInSet(codeLines, listStartLine) || lineInSet(codeLines, listEndLine) {
			return ast.WalkContinue, nil
		}

		blank := quoteBlankLine(f, list, listStartLine)
		quoted := blank != nil
		if needsBlankAdjacent(f, listStartLine, -1, quoted) {
			beforeSet[listStartLine] = blank
		}
		if needsBlankAdjacent(f, listEndLine, +1, quoted) {
			afterSet[listEndLine] = blank
		}

		return ast.WalkContinue, nil
	})

	return beforeSet, afterSet
}

// needsBlankAdjacent returns true if the line adjacent to targetLine
// (direction -1 for before, +1 for after) exists and is non-blank.
// quoted makes a line of only `>` markers count as blank.
func needsBlankAdjacent(f *lint.File, targetLine, direction int, quoted bool) bool {
	adjLine := targetLine + direction
	if adjLine < 1 || adjLine > len(f.Lines) {
		return false
	}
	return !isBlank(f.Lines[adjLine-1], quoted)
}

// FixTitle implements rule.QuickFixTitler.
func (r *Rule) FixTitle() string { return "Add blank lines around list" }

// newlineSep is the bytes.Join separator; a package-level var avoids
// a heap allocation for []byte("\n") on every Fix call.
var newlineSep = []byte("\n")

// enteringKinds is the static node-kind interest CheckNode declares
// via rule.KindScopedChecker; package-level so EnteringKinds returns
// it without allocating.
var enteringKinds = []ast.NodeKind{ast.KindList}

// EnteringKinds implements rule.KindScopedChecker: CheckNode only
// reacts to these node kinds, entering visits only.
func (r *Rule) EnteringKinds() []ast.NodeKind { return enteringKinds }

var _ rule.KindScopedChecker = (*Rule)(nil)
