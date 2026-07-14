package components

import "testing"

func TestEncodeDecodeRoundtrip(t *testing.T) {
	cid := Encode("economy", "balance")
	if cid != "economy::balance" {
		t.Fatalf("encode = %q", cid)
	}
	domain, action, rest := Decode(cid)
	if domain != "economy" || action != "balance" || len(rest) != 0 {
		t.Fatalf("decode = (%q,%q,%v)", domain, action, rest)
	}
}

func TestDecodeWithExtraParts(t *testing.T) {
	domain, action, rest := Decode(Encode("economy", "give", "123", "456"))
	if domain != "economy" || action != "give" || len(rest) != 2 || rest[0] != "123" || rest[1] != "456" {
		t.Fatalf("decode with extra = (%q,%q,%v)", domain, action, rest)
	}
}
