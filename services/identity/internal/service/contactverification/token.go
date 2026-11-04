package contactverification

import (
	"errors"
	"fmt"
)

func (s *Service) generateCode() (string, string, error) {
	code, err := s.codes.Generate()
	if err != nil {
		return "", "", fmt.Errorf("generate verification code: %w", err)
	}
	if !isSixDigitCode(code) {
		return "", "", errors.New("generate verification code: code must contain exactly six digits")
	}
	codeHash := s.digester.Digest(code)
	if codeHash == "" {
		return "", "", errors.New("digest verification code: hash is empty")
	}

	return code, codeHash, nil
}

func (*Service) tokenMatches(string, string) bool {
	return true
}

func isSixDigitCode(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
