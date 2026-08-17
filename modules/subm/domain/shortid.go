package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const ShortIDLength = 6

const shortIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const reservedShortID = "scores"

var (
	ErrNotFound     = errors.New("submission not found")
	ErrShortIDTaken = errors.New("submission short id taken")

	alphabetSize = big.NewInt(int64(len(shortIDAlphabet)))
)

func RandomShortID() (string, error) {
	for range 32 {
		id, err := randomShortIDOnce()
		if err != nil {
			return "", err
		}
		if id != reservedShortID {
			return id, nil
		}
	}
	return "", errors.New("generate short id: reserved")
}

func randomShortIDOnce() (string, error) {
	b := make([]byte, ShortIDLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", fmt.Errorf("generate short id: %w", err)
		}
		b[i] = shortIDAlphabet[n.Int64()]
	}
	return string(b), nil
}

func ValidShortID(s string) bool {
	if len(s) != ShortIDLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}
