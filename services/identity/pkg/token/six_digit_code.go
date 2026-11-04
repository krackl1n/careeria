package token

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
)

const verificationCodeDigits = 6

type SixDigitCode struct {
	random io.Reader
}

func NewSixDigitCode() *SixDigitCode {
	return &SixDigitCode{random: cryptorand.Reader}
}

func newSixDigitCode(random io.Reader) (*SixDigitCode, error) {
	if random == nil {
		return nil, errors.New("token: nil random source")
	}
	return &SixDigitCode{random: random}, nil
}

func (g *SixDigitCode) Generate() (string, error) {
	code := make([]byte, verificationCodeDigits)
	for index := range code {
		digit, err := cryptorand.Int(g.random, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("token: generate verification code: %w", err)
		}
		code[index] = byte('0' + digit.Int64())
	}
	return string(code), nil
}

var _ Generator = (*SixDigitCode)(nil)
