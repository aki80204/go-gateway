package router

import (
	"os"

	"github.com/aki80204/go-gateway/proxy"
	"github.com/aki80204/go-gateway/utils"
	"github.com/aws/aws-lambda-go/events"
)

type ProxyFunc func(events.APIGatewayV2HTTPRequest, string, string) (events.APIGatewayProxyResponse, error)

type Router struct {
	proxy ProxyFunc
}

func NewRouter(pf ProxyFunc) *Router {
	if pf == nil {
		pf = proxy.ProxyRequest
	}
	return &Router{proxy: pf}
}

const (
	ACCOUNT_SERVICE_PATH = "/api/customers/account"
	ASSET_SERVICE_PATH   = "/api/customers/asset"
	BALANCE_SERVICE_PATH = "/api/customers/balance"
	GET                  = "GET"
	POST                 = "POST"
	DELETE               = "DELETE"
	PUT                  = "PUT"
)

// Route は path 毎、HTTP メソッドごとのルーティング処理を行う
func (r *Router) Route(request events.APIGatewayV2HTTPRequest, sub string) (events.APIGatewayProxyResponse, error) {
	switch request.RawPath {
	case ACCOUNT_SERVICE_PATH:
		return r.accountServiceRouter(request, sub)
	case ASSET_SERVICE_PATH:
		return r.assetServiceRouter(request, sub)
	case BALANCE_SERVICE_PATH:
		return r.balanceServiceRouter(request, sub)
	default:
		return utils.ErrorResponse(404, "Not Found"), nil
	}
}

// 顧客管理サービスへのルーティング処理
func (r *Router) accountServiceRouter(request events.APIGatewayV2HTTPRequest, sub string) (events.APIGatewayProxyResponse, error) {
	switch request.RequestContext.HTTP.Method {
	case GET, PUT, DELETE, POST:
		return r.proxy(request, os.Getenv("ACCOUNT_SERVICE_URL"), sub)
	default:
		return utils.ErrorResponse(404, "Not Found"), nil
	}
}

// 資産管理サービスへのルーティング処理
func (r *Router) assetServiceRouter(request events.APIGatewayV2HTTPRequest, sub string) (events.APIGatewayProxyResponse, error) {
	switch request.RequestContext.HTTP.Method {
	case GET, PUT, DELETE, POST:
		return r.proxy(request, os.Getenv("ASSET_SERVICE_URL"), sub)
	default:
		return utils.ErrorResponse(404, "Not Found"), nil
	}
}

// 残高管理サービスへのルーティング処理
func (r *Router) balanceServiceRouter(request events.APIGatewayV2HTTPRequest, sub string) (events.APIGatewayProxyResponse, error) {
	switch request.RequestContext.HTTP.Method {
	case GET, PUT, DELETE, POST:
		return r.proxy(request, os.Getenv("BALANCE_SERVICE_URL"), sub)
	default:
		return utils.ErrorResponse(404, "Not Found"), nil
	}
}
