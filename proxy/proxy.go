package proxy

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aki80204/go-gateway/utils"
	"github.com/aws/aws-lambda-go/events"
)

func ProxyRequest(request events.APIGatewayV2HTTPRequest, targetBaseURL string, sub string) (events.APIGatewayProxyResponse, error) {
	if targetBaseURL == "" {
		return utils.ErrorResponse(500, "Backend service URL not configured"), nil
	}

	// リクエストの組み立て
	targetURL := targetBaseURL + request.RawPath
	req, err := http.NewRequest(request.RequestContext.HTTP.Method, targetURL, strings.NewReader(request.Body))
	if err != nil {
		return utils.ErrorResponse(500, "Internal Proxy Error"), nil
	}

	// ヘッダーの移送と認証情報の付与
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Auth-User-ID", sub)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Lambda Call Error: %v", err)
		return utils.ErrorResponse(502, "Bad Gateway"), nil
	}
	defer resp.Body.Close()

	log.Printf("✅ Received Status: %d", resp.StatusCode) // ここがログに出るか確認！

	// io.ReadAll の前に「最大読み込みサイズ」を制限する（保険）
	// また、読み込みに時間がかかりすぎるのを防ぐ
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 最大1MB
	if err != nil {
		log.Printf("❌ Body Read Error: %v", err)
		// ここでエラーが出るなら、Java側のレスポンス形式が壊れています
	}

	log.Printf("✅ Body Read Success. Size: %d", len(body))

	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}
