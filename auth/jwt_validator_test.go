package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// テスト用の RSA 鍵ペアを生成
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("鍵の生成に失敗しました: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

// JWKS レスポンスを生成
func generateJWKSResponse(publicKey *rsa.PublicKey, kid string) ([]byte, error) {
	// RSA 公開鍵の modulus と exponent を base64url エンコード
	nBytes := publicKey.N.Bytes()
	// 先頭に 0 バイトがある場合は除去（big.Int の Bytes() は符号なし整数を返すため）
	if len(nBytes) > 0 && nBytes[0] == 0 {
		nBytes = nBytes[1:]
	}
	nBase64 := base64.RawURLEncoding.EncodeToString(nBytes)

	// exponent は通常 65537 (0x10001) なので、直接エンコード
	eBytes := big.NewInt(int64(publicKey.E)).Bytes()
	if len(eBytes) > 0 && eBytes[0] == 0 {
		eBytes = eBytes[1:]
	}
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"n":   nBase64,
				"e":   eBase64,
			},
		},
	}
	return json.Marshal(jwks)
}

// テスト用の JWT トークンを生成
func generateTestToken(t *testing.T, privateKey *rsa.PrivateKey, issuer, audience string, expiresIn time.Duration) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": "test-user-123",
		"iat": now.Unix(),
		"exp": now.Add(expiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid-1"

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("トークンの署名に失敗しました: %v", err)
	}
	return tokenString
}

func TestNewValidator(t *testing.T) {
	tests := []struct {
		name          string
		auth0Domain   string
		auth0Audience string
		wantError     bool
		errorContains string
	}{
		{
			name:          "正常系: 環境変数が設定されている",
			auth0Domain:   "test-domain.auth0.com",
			auth0Audience: "https://api.example.com",
			wantError:     false,
		},
		{
			name:          "正常系: https:// プレフィックス付きドメイン",
			auth0Domain:   "https://test-domain.auth0.com",
			auth0Audience: "https://api.example.com",
			wantError:     false,
		},
		{
			name:          "正常系: ドメイン末尾のスラッシュを除去",
			auth0Domain:   "https://test-domain.auth0.com/",
			auth0Audience: "https://api.example.com",
			wantError:     false,
		},
		{
			name:          "エラー: AUTH0_DOMAIN が空",
			auth0Domain:   "",
			auth0Audience: "https://api.example.com",
			wantError:     true,
			errorContains: "AUTH0_DOMAIN",
		},
		{
			name:          "エラー: AUTH0_AUDIENCE が空",
			auth0Domain:   "test-domain.auth0.com",
			auth0Audience: "",
			wantError:     true,
			errorContains: "AUTH0_AUDIENCE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を保存
			oldDomain := os.Getenv("AUTH0_DOMAIN")
			oldAudience := os.Getenv("AUTH0_AUDIENCE")
			defer func() {
				_ = os.Setenv("AUTH0_DOMAIN", oldDomain)
				_ = os.Setenv("AUTH0_AUDIENCE", oldAudience)
			}()

			// テスト用の環境変数を設定
			_ = os.Setenv("AUTH0_DOMAIN", tt.auth0Domain)
			_ = os.Setenv("AUTH0_AUDIENCE", tt.auth0Audience)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			validator, err := NewValidator(ctx)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewValidator() エラーが期待されましたが、nil が返されました")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("NewValidator() エラー = %v, 期待値に含まれるべき文字列 = %v", err.Error(), tt.errorContains)
				}
				if validator != nil {
					t.Errorf("NewValidator() validator = %v, 期待値 = nil", validator)
				}
			} else {
				// JWKS の取得に失敗する可能性があるため、エラーは許容する
				// （実際の Auth0 エンドポイントに接続できない場合）
				if err != nil {
					// JWKS 取得エラーは許容（テスト環境では実際の Auth0 に接続できない可能性がある）
					if !contains(err.Error(), "JWKS の取得に失敗しました") {
						t.Errorf("NewValidator() 予期しないエラー = %v", err)
					}
					return
				}
				if validator == nil {
					t.Errorf("NewValidator() validator = nil, 期待値 = 非 nil")
					return
				}
				if validator.issuer == "" {
					t.Errorf("NewValidator() issuer が空です")
				}
				if validator.audience == "" {
					t.Errorf("NewValidator() audience が空です")
				}
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	// テスト用の鍵ペアを生成
	privateKey, publicKey := generateTestKeyPair(t)

	// モック JWKS サーバーを起動
	kid := "test-kid-1"
	jwksResponse, err := generateJWKSResponse(publicKey, kid)
	if err != nil {
		t.Fatalf("JWKS レスポンスの生成に失敗しました: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jwksResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// テスト用の環境変数を設定
	issuer := mockServer.URL + "/"
	audience := "https://api.example.com"

	_ = os.Setenv("AUTH0_DOMAIN", mockServer.URL)
	_ = os.Setenv("AUTH0_AUDIENCE", audience)
	defer func() {
		_ = os.Unsetenv("AUTH0_DOMAIN")
		_ = os.Unsetenv("AUTH0_AUDIENCE")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	validator, err := NewValidator(ctx)
	if err != nil {
		t.Fatalf("NewValidator() の初期化に失敗しました: %v", err)
	}

	tests := []struct {
		name      string
		tokenFunc func() string
		wantError bool
		errorMsg  string
	}{
		{
			name: "正常系: 有効なトークン",
			tokenFunc: func() string {
				return generateTestToken(t, privateKey, issuer, audience, 1*time.Hour)
			},
			wantError: false,
		},
		{
			name: "エラー: 空のトークン",
			tokenFunc: func() string {
				return ""
			},
			wantError: true,
			errorMsg:  "トークンが空です",
		},
		{
			name: "エラー: 期限切れトークン",
			tokenFunc: func() string {
				return generateTestToken(t, privateKey, issuer, audience, -1*time.Hour)
			},
			wantError: true,
			errorMsg:  "expired", // jwt.Parse が先に期限切れを検出するため
		},
		{
			name: "エラー: 不正な issuer",
			tokenFunc: func() string {
				return generateTestToken(t, privateKey, "https://wrong-issuer.com/", audience, 1*time.Hour)
			},
			wantError: true,
			errorMsg:  "issuerが不正です",
		},
		{
			name: "エラー: 不正な audience (文字列)",
			tokenFunc: func() string {
				return generateTestToken(t, privateKey, issuer, "https://wrong-audience.com", 1*time.Hour)
			},
			wantError: true,
			errorMsg:  "audienceが不正です",
		},
		{
			name: "エラー: 不正な署名",
			tokenFunc: func() string {
				// 別の鍵で署名したトークン
				wrongKey, _ := generateTestKeyPair(t)
				return generateTestToken(t, wrongKey, issuer, audience, 1*time.Hour)
			},
			wantError: true,
			errorMsg:  "トークンのパースに失敗しました",
		},
		{
			name: "エラー: 不正なトークン形式",
			tokenFunc: func() string {
				return "invalid.token.string"
			},
			wantError: true,
			errorMsg:  "トークンのパースに失敗しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.tokenFunc()

			claims, err := validator.ValidateToken(tokenString)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateToken() エラーが期待されましたが、nil が返されました")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateToken() エラー = %v, 期待値に含まれるべき文字列 = %v", err.Error(), tt.errorMsg)
				}
				if claims != nil {
					t.Errorf("ValidateToken() claims = %v, 期待値 = nil", claims)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateToken() エラー = %v, 期待値 = nil", err)
					return
				}
				if claims == nil {
					t.Errorf("ValidateToken() claims = nil, 期待値 = 非 nil")
					return
				}
				// クレームの検証
				if sub, ok := claims["sub"].(string); !ok || sub == "" {
					t.Errorf("ValidateToken() sub クレームが不正です: %v", claims["sub"])
				}
			}
		})
	}
}

func TestValidateToken_AudienceArray(t *testing.T) {
	// テスト用の鍵ペアを生成
	privateKey, publicKey := generateTestKeyPair(t)

	// モック JWKS サーバーを起動
	kid := "test-kid-1"
	jwksResponse, err := generateJWKSResponse(publicKey, kid)
	if err != nil {
		t.Fatalf("JWKS レスポンスの生成に失敗しました: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jwksResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	issuer := mockServer.URL + "/"
	audience := "https://api.example.com"

	_ = os.Setenv("AUTH0_DOMAIN", mockServer.URL)
	_ = os.Setenv("AUTH0_AUDIENCE", audience)
	defer func() {
		_ = os.Unsetenv("AUTH0_DOMAIN")
		_ = os.Unsetenv("AUTH0_AUDIENCE")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	validator, err := NewValidator(ctx)
	if err != nil {
		t.Fatalf("NewValidator() の初期化に失敗しました: %v", err)
	}

	// audience が配列形式のトークンを生成
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": []interface{}{"https://other-api.com", audience}, // 配列形式
		"sub": "test-user-123",
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("トークンの署名に失敗しました: %v", err)
	}

	// 検証
	resultClaims, err := validator.ValidateToken(tokenString)
	if err != nil {
		t.Errorf("ValidateToken() エラー = %v, 期待値 = nil (配列形式の audience は有効であるべき)", err)
		return
	}

	if resultClaims == nil {
		t.Errorf("ValidateToken() claims = nil, 期待値 = 非 nil")
		return
	}

	if sub, ok := resultClaims["sub"].(string); !ok || sub != "test-user-123" {
		t.Errorf("ValidateToken() sub = %v, 期待値 = test-user-123", sub)
	}
}

// ヘルパー関数: 文字列が含まれているかチェック
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
