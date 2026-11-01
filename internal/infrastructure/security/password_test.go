package security

import "testing"

func TestBcryptPasswordHashing(t *testing.T) {
	h, err := NewBcryptPasswordHasher()
	if err != nil {
		t.Fatal(err)
	}
	password := "a-private-password"
	hash, err := h.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	if hash == password || !h.Matches(hash, password) || h.Matches(hash, "wrong") {
		t.Fatal("incorrect password hashing or comparison")
	}
	if h.Matches("", "dummy-password-for-timing-comparison") || h.Matches("invalid hash", password) {
		t.Fatal("missing or invalid hash accepted")
	}
}
