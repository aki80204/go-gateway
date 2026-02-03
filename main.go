package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aki80204/go-gateway/auth"
)

var validator *auth.Validator

// 起動時にAuth0のvalidatorを初期化する
func init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v, err := auth.NewValidator(ctx)
	if err != nil {
		log.Printf("auth validator の初期化に失敗しました: %v", err)
		return
	}
	validator = v
}

// APIGatewayから呼び出されるLambda関数
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// validatorが初期化されていない場合はエラーを返す
	if validator == nil {
		log.Printf("auth validator が初期化されていません。環境変数 AUTH0_DOMAIN/AUTH0_AUDIENCE を確認してください。")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	// Authorizationヘッダーの抽出
	authHeader := request.Headers["Authorization"]
	if authHeader == "" {
		authHeader = request.Headers["authorization"]
	}

	tokenString, err := auth.ExtractBearerToken(authHeader)
	if err != nil {
		log.Printf("Authorizationヘッダーが不正です: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	claims, err := validator.ValidateToken(tokenString)
	if err != nil {
		log.Printf("JWTの検証に失敗しました: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
		}, nil
	}

	sub, _ := claims["sub"].(string)

	resp := map[string]interface{}{
		"message": "Auth0 validation successful.",
		"sub":     sub,
	}

	body, err := json.Marshal(resp)
	if err != nil {
		log.Printf("レスポンスJSONの生成に失敗しました: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            string(body),
		IsBase64Encoded: false,
	}, nil
}

func main() {
	lambda.Start(Handler)
}
