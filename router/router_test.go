package router

import (
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// mockProxyRequest は proxy.ProxyRequest のモック
func mockProxyRequest(request events.APIGatewayV2HTTPRequest, targetBaseURL string, sub string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"message":"mock response"}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

// makeRequest はテスト用の APIGatewayV2HTTPRequest を生成するヘルパー
func makeRequest(path, method string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RawPath: path,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
		Body:    "",
		Headers: map[string]string{},
	}
}

func TestRouter(t *testing.T) {
	r := NewRouter(mockProxyRequest)

	// 環境変数を設定（テスト後に復元）
	origAccountURL := os.Getenv("ACCOUNT_SERVICE_URL")
	origAssetURL := os.Getenv("ASSET_SERVICE_URL")
	origBalanceURL := os.Getenv("BALANCE_SERVICE_URL")
	defer func() {
		os.Setenv("ACCOUNT_SERVICE_URL", origAccountURL)
		os.Setenv("ASSET_SERVICE_URL", origAssetURL)
		os.Setenv("BALANCE_SERVICE_URL", origBalanceURL)
	}()
	os.Setenv("ACCOUNT_SERVICE_URL", "https://account.example.com")
	os.Setenv("ASSET_SERVICE_URL", "https://asset.example.com")
	os.Setenv("BALANCE_SERVICE_URL", "https://balance.example.com")

	tests := []struct {
		name           string
		request        events.APIGatewayV2HTTPRequest
		sub            string
		wantStatusCode int
	}{
		{
			name:           "正常系: Account サービスへ GET ルーティング",
			request:        makeRequest(ACCOUNT_SERVICE_PATH, GET),
			sub:            "user-123",
			wantStatusCode: 200,
		},
		{
			name:           "正常系: Account サービスへ POST ルーティング",
			request:        makeRequest(ACCOUNT_SERVICE_PATH, POST),
			sub:            "user-456",
			wantStatusCode: 200,
		},
		{
			name:           "正常系: Asset サービスへ GET ルーティング",
			request:        makeRequest(ASSET_SERVICE_PATH, GET),
			sub:            "user-789",
			wantStatusCode: 200,
		},
		{
			name:           "正常系: Balance サービスへ GET ルーティング",
			request:        makeRequest(BALANCE_SERVICE_PATH, GET),
			sub:            "user-abc",
			wantStatusCode: 200,
		},
		{
			name:           "異常系: 未知のパスは 404",
			request:        makeRequest("/api/unknown", GET),
			sub:            "user-xyz",
			wantStatusCode: 404,
		},
		{
			name:           "異常系: 空パスは 404",
			request:        makeRequest("", GET),
			sub:            "user-xyz",
			wantStatusCode: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := r.Route(tt.request, tt.sub)

			if err != nil {
				t.Errorf("Router() error = %v, want nil", err)
				return
			}
			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Router() StatusCode = %d, want %d", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestAccountServiceRouter_UnsupportedMethod(t *testing.T) {
	r := NewRouter(mockProxyRequest)

	origURL := os.Getenv("ACCOUNT_SERVICE_URL")
	os.Setenv("ACCOUNT_SERVICE_URL", "https://account.example.com")
	defer func() { os.Setenv("ACCOUNT_SERVICE_URL", origURL) }()

	// サポート外のメソッド（例: PATCH）は 404 を返す
	req := makeRequest(ACCOUNT_SERVICE_PATH, "PATCH")
	resp, err := r.Route(req, "user-123")

	if err != nil {
		t.Errorf("Router() error = %v, want nil", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Router() StatusCode = %d, want 404 for unsupported method", resp.StatusCode)
	}
}

func TestRouter_MockInvocation(t *testing.T) {
	// モックが呼ばれたか検証するために、呼び出し引数を記録
	var capturedURL, capturedSub string
	mock := func(req events.APIGatewayV2HTTPRequest, targetBaseURL string, sub string) (events.APIGatewayProxyResponse, error) {
		capturedURL = targetBaseURL
		capturedSub = sub
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: "{}"}, nil
	}
	r := NewRouter(mock)

	os.Setenv("ACCOUNT_SERVICE_URL", "https://account-svc.test")
	req := makeRequest(ACCOUNT_SERVICE_PATH, GET)

	r.Route(req, "sub-999")

	if capturedURL != "https://account-svc.test" {
		t.Errorf("proxy に渡された URL = %q, want %q", capturedURL, "https://account-svc.test")
	}
	if capturedSub != "sub-999" {
		t.Errorf("proxy に渡された sub = %q, want %q", capturedSub, "sub-999")
	}
}
