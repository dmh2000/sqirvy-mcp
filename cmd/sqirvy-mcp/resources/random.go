package resources

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const (
	// Define the set of allowed characters (alphanumeric)
	allowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// Define the maximum allowed length for random data generation
	maxRandomDataLength = 1024
)

// RandomData generates a cryptographically secure random string of alphanumeric characters
// (a-z, A-Z, 0-9) of the specified length.
// Returns an error if length <= 0, length exceeds maxRandomDataLength, or if generating random indices fails.
func RandomData(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}
	if length > maxRandomDataLength {
		return "", fmt.Errorf("requested length %d exceeds maximum allowed length %d", length, maxRandomDataLength)
	}

	result := make([]byte, length)
	numChars := big.NewInt(int64(len(allowedChars)))

	for i := 0; i < length; i++ {
		// Generate a random index within the bounds of the allowed character set
		randomIndex, err := rand.Int(rand.Reader, numChars)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		// Select the character at the random index
		result[i] = allowedChars[randomIndex.Int64()]
	}

	return string(result), nil
}
