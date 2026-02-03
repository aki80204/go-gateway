package auth

import (
	"errors"
	"strings"
)

// "Bearer <token>"形式のAuthorizationヘッダーからトークンを取り出す
func ExtractBearerToken(header string) (string, error) {
	if strings.TrimSpace(header) == "" {
		return "", errors.New("authorization ヘッダーが存在しません")
	}

	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization ヘッダーが Bearer <token> 形式ではありません")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("authorization ヘッダーが Bearer <token> 形式ではありません")
	}

	return token, nil
}
