package utility

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Hash generates a bcrypt hash of the given plain-text password.
func Hash(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(passwordHash), nil
}

// CompareHashAndPlain compares a bcrypt hash with a plain-text password.
// Returns true if the password matches the hash, false if it doesn't,
// or an error if the comparison fails for other reasons.
func CompareHashAndPlain(hash, plain string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
