package proxy

import (
	"io"
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
		return utils.ErrorResponse(502, "Bad Gateway"), nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(resp.Body)
	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}
