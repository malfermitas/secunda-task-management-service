package app

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenIssuer issues access tokens for authenticated users.
type TokenIssuer interface {
	Issue(userID int64) (string, error)
}
