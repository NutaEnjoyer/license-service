package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateLicenseKey(n int) (string, error) {
	if n <= 0 {
		n = 16 // значение по умолчанию
	}
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Преобразуем байты в строку (hex)
	key := hex.EncodeToString(bytes)

	// Форматируем в виде XXXX-XXXX-XXXX...
	formatted := ""
	for i := 0; i < len(key); i += 4 {
		if i > 0 {
			formatted += "-"
		}
		end := i + 4
		if end > len(key) {
			end = len(key)
		}
		formatted += key[i:end]
	}

	return formatted, nil
}