package auth

import "golang.org/x/crypto/bcrypt"

// BcryptHasher hashes and verifies passwords with bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a bcrypt hasher. A zero cost uses the library default.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash returns a bcrypt hash for the given password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare checks a password against a bcrypt hash.
func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
