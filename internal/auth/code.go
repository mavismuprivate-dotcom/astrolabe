package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func defaultCodeGenerator() (string, error) {
	const max = 1000000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
