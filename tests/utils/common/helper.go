package common

import (
	"crypto/rand"
	"math/big"
	"os"
)

func ReadENVWithDefaultValue(envName string, fallback string) string {
	if os.Getenv(envName) != "" {
		return os.Getenv(envName)
	}
	return fallback
}

func RandomInt(max int) int {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(val.Int64())
}
