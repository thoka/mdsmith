package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripHeadingAttributes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Hand-off {#JbWv0EAr}", "Hand-off"},
		{"Hand-off\t{#h .note data-x=\"1\"}", "Hand-off"},
		{"Hand-off {.note}", "Hand-off"},
		{"Plain", "Plain"},
		{"Notes {draft}", "Notes {draft}"},
		{"{id}: {name}", "{id}: {name}"},
		{"Set {a} to {#x}", "Set {a} to"},
		{"Open {#x} tail", "Open {#x} tail"},
		{"Brace }", "Brace }"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, StripHeadingAttributes(c.in), c.in)
	}
}
