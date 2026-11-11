package catalog

import "testing"

func TestAddressSuggestionLinksToSearch(t *testing.T) {
	t.Parallel()
	got := suggestionHref(SuggestionRow{Type: "address", Label: "Amir Temur ko'chasi 15"})
	want := "/search?q=Amir+Temur+ko%27chasi+15"
	if got != want {
		t.Fatalf("suggestionHref = %q, want %q", got, want)
	}
}
