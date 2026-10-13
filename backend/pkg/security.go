package pkg

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

func Token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func ID() string           { return Token()[:32] }
func Hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func Code() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	return fmtCode(n.Int64())
}
func fmtCode(n int64) string {
	b := [6]byte{}
	for i := 5; i >= 0; i-- {
		b[i] = byte(n%10) + '0'
		n /= 10
	}
	return string(b[:])
}
