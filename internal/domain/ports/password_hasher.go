package ports


type PasswordHasher interface {
	Hash(string) (string, error)
	Matches(hash, password string) bool
}
