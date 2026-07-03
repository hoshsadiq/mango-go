package auth

import "golang.org/x/crypto/bcrypt"

const defaultHashCost = 14

func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, defaultHashCost)
}

func HashPasswordWithCost(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with a stored bcrypt hash.
// It returns true if the password matches the hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
