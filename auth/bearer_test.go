package auth

import (
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantError string
	}{
		{
			name:      "正常系: Bearer トークン（大文字）",
			header:    "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantError: "",
		},
		{
			name:      "正常系: bearer トークン（小文字）",
			header:    "bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantError: "",
		},
		{
			name:      "正常系: BEARER トークン（すべて大文字）",
			header:    "BEARER eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantError: "",
		},
		{
			name:      "正常系: 前後にスペースがある",
			header:    "  Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9  ",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantError: "",
		},
		{
			name:      "正常系: Bearer とトークンの間に複数のスペース",
			header:    "Bearer   eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantError: "",
		},
		{
			name:      "エラー: 空文字列",
			header:    "",
			wantToken: "",
			wantError: "authorization ヘッダーが存在しません",
		},
		{
			name:      "エラー: スペースのみ",
			header:    "   ",
			wantToken: "",
			wantError: "authorization ヘッダーが存在しません",
		},
		{
			name:      "エラー: Bearer だけ（トークンがない）",
			header:    "Bearer",
			wantToken: "",
			wantError: "authorization ヘッダーが Bearer <token> 形式ではありません",
		},
		{
			name:      "エラー: Bearer の後にスペースのみ",
			header:    "Bearer ",
			wantToken: "",
			wantError: "authorization ヘッダーが Bearer <token> 形式ではありません",
		},
		{
			name:      "エラー: Bearer の後に複数のスペースのみ",
			header:    "Bearer    ",
			wantToken: "",
			wantError: "authorization ヘッダーが Bearer <token> 形式ではありません",
		},
		{
			name:      "エラー: 3つ以上の部分に分割される",
			header:    "Bearer token1 token2",
			wantToken: "",
			wantError: "authorization ヘッダーが Bearer <token> 形式ではありません",
		},
		{
			name:      "エラー: Basic 認証",
			header:    "Basic dXNlcm5hbWU6cGFzc3dvcmQ=",
			wantToken: "",
			wantError: "authorization ヘッダーが Bearer <token> 形式ではありません",
		},
		{
			name:      "エラー: プレフィックスなし",
			header:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "",
			wantError: "authorization ヘッダーが Bearer <token> 形式ではありません",
		},
		{
			name:      "正常系: 長いトークン",
			header:    "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			wantToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractBearerToken(tt.header)

			if tt.wantError != "" {
				if err == nil {
					t.Errorf("ExtractBearerToken() エラーが期待されましたが、nil が返されました")
					return
				}
				if err.Error() != tt.wantError {
					t.Errorf("ExtractBearerToken() エラー = %v, 期待値 = %v", err.Error(), tt.wantError)
				}
				if token != "" {
					t.Errorf("ExtractBearerToken() トークン = %v, 期待値 = \"\"", token)
				}
			} else {
				if err != nil {
					t.Errorf("ExtractBearerToken() エラー = %v, 期待値 = nil", err)
					return
				}
				if token != tt.wantToken {
					t.Errorf("ExtractBearerToken() トークン = %v, 期待値 = %v", token, tt.wantToken)
				}
			}
		})
	}
}
