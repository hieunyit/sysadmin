package passwordutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	lowerCharset  = "abcdefghijkmnopqrstuvwxyz"
	upperCharset  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitCharset  = "23456789"
	symbolCharset = "!@#$%*-_"
)

func GenerateTemporary() (string, error) {
	const passwordLength = 14

	all := lowerCharset + upperCharset + digitCharset + symbolCharset
	password := make([]byte, 0, passwordLength)

	requiredSets := []string{
		lowerCharset,
		upperCharset,
		digitCharset,
		symbolCharset,
	}
	for _, charset := range requiredSets {
		ch, err := randomChar(charset)
		if err != nil {
			return "", err
		}
		password = append(password, ch)
	}
	for len(password) < passwordLength {
		ch, err := randomChar(all)
		if err != nil {
			return "", err
		}
		password = append(password, ch)
	}
	if err := shuffle(password); err != nil {
		return "", err
	}
	return string(password), nil
}

func randomChar(charset string) (byte, error) {
	if len(charset) == 0 {
		return 0, fmt.Errorf("charset is empty")
	}
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, fmt.Errorf("generate random index failed: %w", err)
	}
	return charset[idx.Int64()], nil
}

func shuffle(in []byte) error {
	for i := len(in) - 1; i > 0; i-- {
		jRaw, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("shuffle random index failed: %w", err)
		}
		j := int(jRaw.Int64())
		in[i], in[j] = in[j], in[i]
	}
	return nil
}
