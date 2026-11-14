package entities

type Admin struct {
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}
