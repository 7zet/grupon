package catalog

import (
	"strings"
	"testing"
)

func TestExpandSearch(t *testing.T) {
	t.Parallel()
	for _, query := range []string{"masaj", "массаж", "масаж", "Massage"} {
		got := ExpandSearch(query)
		for _, want := range []string{"massaj", "массаж", "massage"} {
			if !strings.Contains(strings.ToLower(got), want) {
				t.Errorf("ExpandSearch(%q) = %q, missing %q", query, got, want)
			}
		}
	}
	if got := ExpandSearch("something else"); got != "something else" {
		t.Errorf("unrelated query changed to %q", got)
	}
}
