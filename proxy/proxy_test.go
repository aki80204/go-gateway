package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func makeRequest(path, method, body string, headers map[string]string) events.APIGatewayV2HTTPRequest {
	if headers == nil {
		headers = map[string]string{}
	}
	return events.APIGatewayV2HTTPRequest{
		RawPath: path,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
		Body:    body,
		Headers: headers,
	}
}

func TestProxyRequest_EmptyBaseURL(t *testing.T) {
	req := makeRequest("/api/test", "GET", "", nil)
	resp, err := ProxyRequest(req, "", "user-123")

	if err != nil {
		t.Errorf("ProxyRequest() error = %v, want nil", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("ProxyRequest() StatusCode = %d, want 500", resp.StatusCode)
	}
	if resp.Body != `{"error":"Backend service URL not configured"}` {
		t.Errorf("ProxyRequest() Body = %q, want error message", resp.Body)
	}
}

func TestProxyRequest_Success(t *testing.T) {
	// モックバックエンドサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"test"}`))
	}))
	defer server.Close()

	req := makeRequest("/api/customers/account", "GET", "", nil)
	resp, err := ProxyRequest(req, server.URL, "sub-123")

	if err != nil {
		t.Errorf("ProxyRequest() error = %v, want nil", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("ProxyRequest() StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.Body != `{"id":1,"name":"test"}` {
		t.Errorf("ProxyRequest() Body = %q, want %q", resp.Body, `{"id":1,"name":"test"}`)
	}
}

func TestProxyRequest_ReturnsBackendStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	req := makeRequest("/api/unknown", "GET", "", nil)
	resp, err := ProxyRequest(req, server.URL, "user-456")

	if err != nil {
		t.Errorf("ProxyRequest() error = %v, want nil", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("ProxyRequest() StatusCode = %d, want 404", resp.StatusCode)
	}
	if resp.Body != `{"error":"not found"}` {
		t.Errorf("ProxyRequest() Body = %q", resp.Body)
	}
}

func TestProxyRequest_ForwardsHeadersAndSetsXAuthUserID(t *testing.T) {
	var capturedPath, capturedMethod, capturedAuthUser string
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		capturedAuthUser = r.Header.Get("X-Auth-User-ID")
		capturedContentType = r.Header.Get("Content-Type")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	req := makeRequest("/api/customers/account", "POST", `{"key":"value"}`, map[string]string{
		"Content-Type": "application/json",
		"X-Custom-Header": "custom-value",
	})
	resp, err := ProxyRequest(req, server.URL, "auth-user-789")

	if err != nil {
		t.Errorf("ProxyRequest() error = %v, want nil", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("ProxyRequest() StatusCode = %d, want 200", resp.StatusCode)
	}
	if capturedPath != "/api/customers/account" {
		t.Errorf("バックエンドへのパス = %q, want /api/customers/account", capturedPath)
	}
	if capturedMethod != "POST" {
		t.Errorf("バックエンドへのメソッド = %q, want POST", capturedMethod)
	}
	if capturedAuthUser != "auth-user-789" {
		t.Errorf("X-Auth-User-ID = %q, want auth-user-789", capturedAuthUser)
	}
	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", capturedContentType)
	}
}

func TestProxyRequest_ForwardsBody(t *testing.T) {
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	reqBody := `{"accountId":"acc-123"}`
	req := makeRequest("/api/customers/account", "POST", reqBody, nil)
	_, err := ProxyRequest(req, server.URL, "user-1")

	if err != nil {
		t.Errorf("ProxyRequest() error = %v, want nil", err)
	}
	if capturedBody != reqBody {
		t.Errorf("バックエンドへのBody = %q, want %q", capturedBody, reqBody)
	}
}

func TestProxyRequest_Returns502OnConnectionFailure(t *testing.T) {
	// リスニングしていないポートへ接続試行 → connection refused
	invalidURL := "http://127.0.0.1:19999"
	req := makeRequest("/api/test", "GET", "", nil)

	resp, err := ProxyRequest(req, invalidURL, "user-1")

	if err != nil {
		t.Errorf("ProxyRequest() error = %v, want nil (always returns nil)", err)
	}
	if resp.StatusCode != 502 {
		t.Errorf("ProxyRequest() StatusCode = %d, want 502 (Bad Gateway)", resp.StatusCode)
	}
	if resp.Body != `{"error":"Bad Gateway"}` {
		t.Errorf("ProxyRequest() Body = %q, want Bad Gateway error", resp.Body)
	}
}
