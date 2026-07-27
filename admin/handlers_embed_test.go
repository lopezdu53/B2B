//nolint:testpackage // exercises unexported embed-token helpers
package admin

import "testing"

func TestEmbedTokenRoundTrip(t *testing.T) {
	secret := []byte("test-secret-key-000000000000000000000000")

	for _, id := range []int64{1, 42, 9999} {
		tok := embedToken(id, secret)

		got, ok := parseEmbedToken(tok, secret)
		if !ok {
			t.Fatalf("valid token for %d rejected", id)
		}

		if got != id {
			t.Fatalf("tenant id: want %d got %d", id, got)
		}
	}
}

func TestEmbedTokenRejectsForgeries(t *testing.T) {
	secret := []byte("test-secret-key-000000000000000000000000")
	tok := embedToken(7, secret)

	cases := map[string]string{
		"tampered signature": tok[:len(tok)-1] + "0",
		"wrong tenant id":    "8." + tok[2:],
		"no dot":             "12345",
		"empty":              "",
		"trailing dot":       "7.",
	}

	for name, bad := range cases {
		if _, ok := parseEmbedToken(bad, secret); ok {
			t.Errorf("%s: forged token accepted (%q)", name, bad)
		}
	}

	// A token minted with a different secret must not validate.
	other := embedToken(7, []byte("a-completely-different-secret-key-000000"))
	if _, ok := parseEmbedToken(other, secret); ok {
		t.Error("token from different secret accepted")
	}
}
