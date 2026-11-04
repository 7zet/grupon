package identity

import "testing"

func TestNewOpaqueTokenIsRandomAndLongEnough(t *testing.T) {
	t.Parallel()

	first, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 43 {
		t.Errorf("token length = %d, want 43 (256 bits, base64url)", len(first))
	}
	if first == second {
		t.Error("two tokens are equal")
	}
}

func TestHashTokenIsSHA256AndNotTheRawValue(t *testing.T) {
	t.Parallel()

	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := HashToken("abc"); got != want {
		t.Fatalf("HashToken(abc) = %s, want %s", got, want)
	}
	if HashToken("abc") == "abc" {
		t.Fatal("HashToken returned the raw value")
	}
}
