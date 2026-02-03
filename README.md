# Go Gateway for AWS Lambda (Auth0 Integration)

AWS Lambda 上で動作する、Go言語製の高性能 API Gateway プロトタイプです。
Auth0 と連携した JWT 認証機能を備え、マイクロサービスの前方配置（門番）としての利用を想定しています。

## 🚀 特徴

* **超高速レスポンス**: Go 1.x / Amazon Linux 2023 ランタイムを採用。実行時間 **1.39ms** という極めて低いレイテンシを実現。
* **セキュアな認証**: Auth0 を ID プロバイダーとして利用。RS256 アルゴリズムによる JWT 署名検証を実装。
* **「現場」仕様の構成**: `Makefile` によるビルド管理、`golangci-lint` による静的解析、`.gitignore` によるクリーンなリポジトリ管理を導入。

## 🛠 アーキテクチャ

1. **Request**: クライアントが Auth0 から取得した JWT を `Authorization` ヘッダーに付与してアクセス。
2. **API Gateway**: リクエストを本 Lambda 関数へルーティング。
3. **Go Lambda (This Repo)**: 
    * Auth0 の `jwks.json` から公開鍵を取得（初回取得後はメモリキャッシュ可能）。
    * JWT の署名・有効期限・Audience を検証。
    * 検証成功後、後続のバックエンドサービス（Java/Spring Boot等）へリクエストを認可。

## 📦 セットアップ

### 1. 依存関係のインストール
```bash
go mod tidy
```

### 2. 環境変数の設定
AWS Lambda の「設定」>「環境変数」タブにて、以下の値を設定してください。

| キー | 説明 | 例 |
| :--- | :--- | :--- |
| `AUTH0_DOMAIN` | Auth0のドメイン（末尾に / を含む） | `https://xxxx.auth0.com/` |
| `AUTH0_AUDIENCE` | API Identifier（識別子） | `https://api.kazuma-exchange.com` |

### 3. ビルドとパッケージング
Makefile を使用して、Lambda 専用バイナリ（bootstrap）の作成と zip 圧縮を一括で行います。

```bash
# ビルド、テスト、Lint、zip圧縮を全て実行
make
```
生成された go-gateway.zip を AWS Lambda コンソールからアップロードしてください。

## 🧪 運用・テスト

### 静的解析 (Lint) の実行
コードの品質を担保するため、`golangci-lint` を使用して構文チェックとバグの未然防止を行います。
```bash
make lint
```

### AWS Lambda での疎通確認
Lambda コンソールのテストイベントにて、以下の JSON 構造（Auth0 で取得した JWT を含む）で動作確認が可能です。

```json
{
  "headers": {
    "Authorization": "Bearer <YOUR_AUTH0_JWT>"
  },
  "httpMethod": "GET",
  "path": "/"
}
```

## 📈 パフォーマンス（実測値）
Go 言語の特性を活かし、極めて低いレイテンシを実現しています。

* **Actual Duration**: 1.39 ms
* **Billed Duration**: 2 ms
* **Max Memory Used**: 32 MB

---

## 📜 ライセンス
[MIT License](LICENSE)
