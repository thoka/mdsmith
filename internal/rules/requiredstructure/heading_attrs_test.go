package requiredstructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A trailing heading attribute list such as "{#id}" is not part of
// the heading text: section-schema.md says the match target strips
// it. These tests pin that on the proto.md path, the inline
// kind-schema path, and the regex matcher.

func TestCheck_ProtoHeadingAttrListStripped(t *testing.T) {
	schemaPath := writeSchema(t, "# ?\n\n## Hand-off\n")
	r := &Rule{Schema: schemaPath}
	f := newTestFile(t, "doc.md",
		"# Plan\n\n## Hand-off {#JbWv0EAr}\n\nText.\n")
	assert.Empty(t, r.Check(f))
}

func TestCheck_ProtoSchemaHeadingAttrListStripped(t *testing.T) {
	schemaPath := writeSchema(t, "# ?\n\n## Hand-off {#handoff}\n")
	r := &Rule{Schema: schemaPath}
	f := newTestFile(t, "doc.md", "# Plan\n\n## Hand-off\n\nText.\n")
	assert.Empty(t, r.Check(f))
}

func TestCheck_ProtoHeadingAttrListClassAndKeyValue(t *testing.T) {
	schemaPath := writeSchema(t, "# ?\n\n## Hand-off\n")
	r := &Rule{Schema: schemaPath}
	f := newTestFile(t, "doc.md",
		"# Plan\n\n## Hand-off {#h .note data-x=\"1\"}\n\nText.\n")
	assert.Empty(t, r.Check(f))
}

func TestCheck_ProtoHeadingBraceTextNotAttrList(t *testing.T) {
	schemaPath := writeSchema(t, "# ?\n\n## Notes\n")
	r := &Rule{Schema: schemaPath}
	f := newTestFile(t, "doc.md", "# Plan\n\n## Notes {draft}\n\nText.\n")
	expectDiagMsg(t, r.Check(f), `## Notes: got <missing>`)
}

func TestCheck_InlineSchemaHeadingAttrListStripped(t *testing.T) {
	r := &Rule{InlineSchema: inlineSchema(t, map[string]any{
		"closed": true,
		"sections": []any{
			map[string]any{"heading": "Hand-off"},
		},
	})}
	f := newTestFile(t, "doc.md",
		"# Plan\n\n## Hand-off {#JbWv0EAr}\n\nText.\n")
	assert.Empty(t, r.Check(f))
}

func TestCheck_InlineSchemaRegexHeadingAttrListStripped(t *testing.T) {
	r := &Rule{InlineSchema: inlineSchema(t, map[string]any{
		"closed": true,
		"sections": []any{
			map[string]any{
				"heading": map[string]any{"regex": "Hand-off"},
			},
		},
	})}
	f := newTestFile(t, "doc.md",
		"# Plan\n\n## Hand-off {#JbWv0EAr}\n\nText.\n")
	assert.Empty(t, r.Check(f))
}

func TestHeadingMatchesLine_AttrList(t *testing.T) {
	h := docHeading{Level: 2, Text: "Goal"}
	assert.True(t, headingMatchesLine(h, "## Goal {#goal}"))
	assert.False(t, headingMatchesLine(h, "## Goal {draft}"))
}
