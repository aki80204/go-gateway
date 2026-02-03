package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	keyfunc  keyfunc.Keyfunc
	issuer   string
	audience string
}

// 環境変数を使用してAuth0のValidatorを初期化する
//
// 必要な環境変数:
//   - AUTH0_DOMAIN  例: "example-region.auth0.com" または "https://example-region.auth0.com"
//   - AUTH0_AUDIENCE (API Identifier)
func NewValidator(ctx context.Context) (*Validator, error) {
	domain := os.Getenv("AUTH0_DOMAIN")
	audience := os.Getenv("AUTH0_AUDIENCE")

	if domain == "" || audience == "" {
		return nil, fmt.Errorf("AUTH0_DOMAIN と AUTH0_AUDIENCE を環境変数に設定してください")
	}

	// ドメインの正規化
	if !strings.HasPrefix(domain, "https://") && !strings.HasPrefix(domain, "http://") {
		domain = "https://" + domain
	}
	domain = strings.TrimRight(domain, "/")

	issuer := domain + "/"
	jwksURL := issuer + ".well-known/jwks.json"

	// keyfunc v3 のデフォルト設定で JWKS を取得する。
	// 以後は 内部キャッシュを使いつつ、自動的にリフレッシュする仕組み
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("JWKS の取得に失敗しました (%s): %w", jwksURL, err)
	}

	return &Validator{
		keyfunc:  kf,
		issuer:   issuer,
		audience: audience,
	}, nil
}

// ValidateToken は渡された JWT 文字列を検証し、有効な場合はクレームを返します。
func (v *Validator) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, errors.New("トークンが空です")
	}

	// keyfunc v3のKeyfuncを使って署名検証を行う。
	token, err := jwt.Parse(tokenString, v.keyfunc.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("トークンのパースに失敗しました: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("トークンが無効です")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("トークンのclaim形式が想定外です")
	}

	// 有効期限の検証
	if exp, err := claims.GetExpirationTime(); err != nil {
		return nil, fmt.Errorf("exp claimの取得に失敗しました: %w", err)
	} else if exp != nil && time.Now().After(exp.Time) {
		return nil, errors.New("トークンの有効期限が切れています")
	}

	// iss検証
	if iss, ok := claims["iss"].(string); !ok || iss != v.issuer {
		return nil, errors.New("issuerが不正です")
	}

	// aud検証 (Auth0では文字列または配列のどちらか)
	audValid := false
	if aud, ok := claims["aud"].(string); ok {
		if aud == v.audience {
			audValid = true
		}
	} else if auds, ok := claims["aud"].([]interface{}); ok {
		for _, a := range auds {
			if s, ok := a.(string); ok && s == v.audience {
				audValid = true
				break
			}
		}
	}
	if !audValid {
		return nil, errors.New("audienceが不正です")
	}

	return claims, nil
}
