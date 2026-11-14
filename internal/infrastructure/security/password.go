package security

import "golang.org/x/crypto/bcrypt"

// BcryptPasswordHasher keeps password algorithms outside application policy.
type BcryptPasswordHasher struct{ dummyHash string }

func NewBcryptPasswordHasher() (*BcryptPasswordHasher, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing-comparison"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &BcryptPasswordHasher{dummyHash: string(dummy)}, nil
}
func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
func (h *BcryptPasswordHasher) Matches(hash, password string) bool {
	unknown := hash == ""
	if unknown {
		hash = h.dummyHash
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil && !unknown
}
