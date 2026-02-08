package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aki80204/go-gateway/auth"
	"github.com/aki80204/go-gateway/proxy"
	"github.com/aki80204/go-gateway/router"
	"github.com/aki80204/go-gateway/utils"
)

var validator *auth.Validator
var gatewayRouter *router.Router

// 起動時にAuth0のvalidatorとRouterを初期化する
func init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v, err := auth.NewValidator(ctx)
	if err != nil {
		log.Printf("auth validator の初期化に失敗しました: %v", err)
		return
	}
	validator = v
	gatewayRouter = router.NewRouter(proxy.ProxyRequest)
}

// APIGatewayから呼び出されるLambda関数
func Handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	// validatorが初期化されていない場合はエラーを返す
	if validator == nil {
		log.Printf("auth validator が初期化されていません。環境変数 AUTH0_DOMAIN/AUTH0_AUDIENCE を確認してください。")
		return utils.ErrorResponse(500, "Internal Server Error"), nil
	}

	sub, err := auth.CheckAuth(*validator, request)
	if err != nil {
		return utils.ErrorResponse(401, "Unauthorized"), nil
	}

	return gatewayRouter.Route(request, sub)
}

func main() {
	lambda.Start(Handler)
}
